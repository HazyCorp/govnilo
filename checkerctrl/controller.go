package checkerctrl

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HazyCorp/govnilo/common/checkersettings"
	"github.com/HazyCorp/govnilo/common/hzlog"
	"github.com/HazyCorp/govnilo/hazycheck"
	"github.com/HazyCorp/govnilo/pkg/raterunner"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	SyncInterval time.Duration `json:"sync_interval" yaml:"sync_interval"`
	TargetHost   string        `json:"target_host" yaml:"target_host"`
}

type Controller struct {
	l                  *slog.Logger
	registeredCheckers map[hazycheck.CheckerID]hazycheck.Checker
	registeredSploits  map[hazycheck.SploitID]hazycheck.Sploit
	storage            ControllerStorage
	settingsProvider   SettingsProvider
	conf               Config
	strategy           SaveStrategy

	runCtx     context.Context
	runCancel  context.CancelFunc
	runErrChan chan error

	checkerMetricsMu sync.RWMutex
	checkerMetrics   map[hazycheck.CheckerID]*checkerMetrics

	sploitMetricsMu sync.RWMutex
	sploitMetrics   map[hazycheck.SploitID]*sploitMetrics

	// BIG TODO!!!
	providersMu     sync.RWMutex
	cachedProviders map[string]hazycheck.Connector

	currentSettings atomic.Pointer[checkersettings.Settings]

	lastDeletionMu sync.RWMutex
	lastDeletion   map[hazycheck.CheckerID]time.Time

	rr *raterunner.RateRunner
}

type ControllerIn struct {
	fx.In

	Logger           *slog.Logger
	Checkers         []hazycheck.Checker `group:"checkers"`
	Sploits          []hazycheck.Sploit  `group:"sploits"`
	Storage          ControllerStorage
	SettingsProvider SettingsProvider
	Config           Config
	Strategy         SaveStrategy
}

func New(in ControllerIn) *Controller {
	idToChecker := make(map[hazycheck.CheckerID]hazycheck.Checker, len(in.Checkers))
	for _, check := range in.Checkers {
		idToChecker[check.CheckerID()] = check
	}

	idToSploit := make(map[hazycheck.SploitID]hazycheck.Sploit, len(in.Sploits))
	for _, sploit := range in.Sploits {
		idToSploit[sploit.SploitID()] = sploit
	}

	return &Controller{
		l:                  in.Logger.With(slog.String("component", "checker-controller")),
		registeredCheckers: idToChecker,
		registeredSploits:  idToSploit,
		storage:            in.Storage,
		rr:                 raterunner.New(in.Logger),
		conf:               in.Config,
		runErrChan:         make(chan error, 1),
		strategy:           in.Strategy,
		settingsProvider:   in.SettingsProvider,
		checkerMetrics:     make(map[hazycheck.CheckerID]*checkerMetrics),
		sploitMetrics:      make(map[hazycheck.SploitID]*sploitMetrics),

		cachedProviders: make(map[string]hazycheck.Connector),
		lastDeletion:    make(map[hazycheck.CheckerID]time.Time),
	}
}

func NewFX(in ControllerIn, lc fx.Lifecycle, sd fx.Shutdowner) *Controller {
	ctrl := New(in)
	lc.Append(fx.Hook{
		OnStart: ctrl.Start,
		OnStop:  ctrl.Stop,
	})

	return ctrl
}

func (c *Controller) Start(ctx context.Context) error {
	// if err := c.syncState(ctx); err != nil {
	// 	return errors.Wrap(err, "cannot initially sync state")
	// }

	c.runCtx, c.runCancel = context.WithCancel(context.Background())
	eg, ctx := errgroup.WithContext(c.runCtx)
	eg.Go(func() error { return c.rr.Run(ctx) })
	eg.Go(func() error { return c.run(ctx) })

	go func() {
		err := eg.Wait()
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			err = nil
		}

		c.runErrChan <- err
	}()

	return nil
}

func (c *Controller) Stop(ctx context.Context) error {
	c.runCancel()

	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "cannot wait for stop of checker controller")
	case err := <-c.runErrChan:
		return err
	}
}

func (c *Controller) run(ctx context.Context) error {
	c.l.Info("checker controller started")
	defer c.l.Info("checker.run routine ended it's job")

	for {
		if err := c.waitFor(ctx, c.conf.SyncInterval); err != nil {
			return err
		}

		if err := c.syncState(ctx); err != nil {
			c.l.Warn(
				"error occurred, while trying to sync checker controller state with storage",
				hzlog.Error(err),
			)
		} else {
			c.l.Info("checker state syncronized")
		}
	}
}

func (c *Controller) genSploitRunAttackTask(
	sploit hazycheck.Sploit,
) raterunner.TaskFunc {
	sploitID := sploit.SploitID()
	m := c.sploitMetricsFor(sploitID)
	l := c.l.With(slog.Any("sploit_id", sploitID))

	return func(ctx context.Context) error {
		currentSettings := c.currentSettings.Load()
		serviceSettings := currentSettings.Services[sploitID.Service]
		if serviceSettings == nil {
			return errors.Errorf("cannot find service %s in current settings", sploitID.Service)
		}

		start := time.Now()

		var sploitErr error
		// we use defer here to recover from panics
		defer func() {
			if r := recover(); r != nil {
				sploitErr = errors.Errorf("sploit paniced: %+v", r)
			}

			if sploitErr != nil {
				m.FailCounter.Inc()
				m.FailDuration.UpdateDuration(start)

				l.DebugContext(ctx, "sploit.RunAttack failed", hzlog.Error(sploitErr))
				return
			}

			m.SuccessCounter.Inc()
			m.SuccessDuration.UpdateDuration(start)
		}()

		sploitErr = sploit.RunAttack(ctx, serviceSettings.Target)

		// don't need to save anything after sploit running
		return nil
	}
}

func (c *Controller) genCheckerCheckTask(
	checker hazycheck.Checker,
) raterunner.TaskFunc {
	checkerID := checker.CheckerID()
	m := c.checkerMetricsFor(checkerID)
	l := c.l.With("checker_id", checkerID)

	return func(ctx context.Context) error {
		currentSettings := c.currentSettings.Load()
		serviceSettings := currentSettings.Services[checkerID.Service]
		if serviceSettings == nil {
			return errors.Errorf("cannot find service %s in current settings", checkerID.Service)
		}

		checkerSettings := serviceSettings.Checkers[checkerID.Name]
		if checkerSettings == nil {
			return errors.Errorf("cannot find checker %s in %s service settings", checkerID.Name, checkerID.Service)
		}

		start := time.Now()
		var data []byte
		var checkErr error
		// we use defer here to recover from panics
		defer func() {
			if r := recover(); r != nil {
				checkErr = hazycheck.InternalError(errors.Errorf("checker check paniced with message: %+v", checkErr))
			}

			success := true
			if checkErr != nil {
				var internalErr *hazycheck.InternalErr
				if errors.As(checkErr, &internalErr) {
					// internal error occured, we cannot give penalties to teams

					l.WarnContext(ctx, "internal checker error occurred", hzlog.Error(internalErr.Internal))

					m.CheckInternalErrorsTotal.Inc()
					m.CheckInternalErrorsDuration.UpdateDuration(start)

					return
				}

				l.Debug(
					"checker.Check errored",
					hzlog.Error(checkErr),
				)

				// if checker.Check fails -- it's OK, it's expected behaviour, so, we don't need to fail this task
				success = false
			}

			if success {
				l.Debug("checker run succeed", slog.Any("checker_id", checker.CheckerID()))
				m.SuccessCheckCounter.Inc()
				m.SuccessCheckDuration.UpdateDuration(start)
				m.SuccessCheckPoints.Add(checkerSettings.Check.SuccessPoints)

				l.Debug("saving the data", slog.String("data", string(data)))
				if err := c.saveCheckerData(ctx, checker.CheckerID(), data); err != nil {
					l.WarnContext(ctx, "cannot save check data to storage", hzlog.Error(err))
					return
				}
			} else {
				m.FailCheckCounter.Inc()
				m.FailCheckDuration.UpdateDuration(start)
				m.FailCheckPenalty.Add(checkerSettings.Check.FailPenalty)
			}
		}()

		data, checkErr = checker.Check(ctx, serviceSettings.Target)

		return nil
	}
}

func (c *Controller) genCheckerGetTask(
	checker hazycheck.Checker,
) raterunner.TaskFunc {
	checkerID := checker.CheckerID()
	m := c.checkerMetricsFor(checkerID)
	l := c.l.With(slog.Any("checker_id", checker.CheckerID()))

	return func(ctx context.Context) error {
		currentSettings := c.currentSettings.Load()
		serviceSettings := currentSettings.Services[checkerID.Service]
		if serviceSettings == nil {
			return errors.Errorf("cannot find service %s in current settings", checkerID.Service)
		}

		checkerSettings := serviceSettings.Checkers[checkerID.Name]
		if checkerSettings == nil {
			return errors.Errorf("cannot find checker %s in %s service settings", checkerID.Name, checkerID.Service)
		}

		pool, err := c.storage.GetCheckerDataPool(ctx, checker.CheckerID())
		if err != nil {
			return errors.Wrap(err, "cannot get pool of data to run Checker.Get")
		}

		if len(pool) == 0 {
			return errors.Errorf("data pool is empty, cannot get data to run Checker.Get")
		}

		start := time.Now()

		var getErr error
		// we use defer to catch panics
		defer func() {
			if r := recover(); r != nil {
				// panic is an internal error
				getErr = hazycheck.InternalError(errors.Errorf("checker.get paniced: %+v", r))
			}

			success := true
			if getErr != nil {
				var internalErr *hazycheck.InternalErr
				if errors.As(getErr, &internalErr) {
					// internal error occurred, we don't need to increment any points or penalties
					l.WarnContext(ctx, "internal error occurred", hzlog.Error(internalErr.Internal))

					m.GetInternalErrorsTotal.Inc()
					m.GetInternalErrorsDuration.UpdateDuration(start)

					return
				}

				// if checker.Get fails -- it's OK, it's expected behaviour, so, we don't need to fail this task
				success = false
				l.Debug("checker.Get of failed with message", hzlog.Error(getErr))
			}

			if success {
				l.Debug("checker.Get succeed")
				m.SuccessGetCounter.Inc()
				m.SuccessGetDuration.UpdateDuration(start)
				m.SuccessGetPoints.Add(checkerSettings.Get.SuccessPoints)
			} else {
				m.FailGetCounter.Inc()
				m.FailGetDuration.UpdateDuration(start)
				m.FailGetPenalty.Add(checkerSettings.Get.FailPenalty)
			}
		}()

		idx := rand.Intn(len(pool))
		data := pool[idx]

		getErr = checker.Get(ctx, serviceSettings.Target, data.Data)

		return nil
	}
}

func (c *Controller) saveCheckerData(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
	data []byte,
) error {
	pool, err := c.storage.GetCheckerDataPool(ctx, checkerID)
	if err != nil {
		return errors.Wrap(err, "cannot get current data pool")
	}

	if c.strategy.NeedSave(uint64(len(pool))) {
		if err := c.storage.AppendCheckerData(ctx, checkerID, data); err != nil {
			return errors.Wrap(err, "cannot append the checker data, when stratedgy said to do that")
		}
	}

	// deletion is a heavy operation, we have to run it not to frequent
	c.lastDeletionMu.RLock()
	lastTime, exists := c.lastDeletion[checkerID]
	c.lastDeletionMu.RUnlock()

	// TODO: use config
	if !exists || time.Since(lastTime) > time.Second*10 {
		// need to delete
		c.storage.RemoveDataFromPool(ctx, checkerID, c.strategy.NeedDelete)

		// save infomration
		c.lastDeletionMu.Lock()
		c.lastDeletion[checkerID] = time.Now()
		c.lastDeletionMu.Unlock()
	}

	return nil
}

func (c *Controller) syncState(ctx context.Context) error {
	c.l.Debug("syncing controller state")

	currentSettings, err := c.settingsProvider.GetSettings(ctx)
	if err != nil {
		return errors.Wrap(
			err,
			"cannot receive routines state from storage",
		)
	}

	// MUSTHAVE!!!!
	currentSettings.NormalizePoints()

	c.currentSettings.Store(currentSettings)

	var errlist *multierror.Error

	for checkerID, checker := range c.registeredCheckers {
		l := c.l.With("checker_id", checkerID)

		svcName, checkerName := checkerID.Service, checkerID.Name

		checkerExists := true
		var svcDesc *checkersettings.ServiceSettings
		var checkerDesc *checkersettings.CheckerSettings

		// using func here to avoid nil reference panics
		// return allows us to skip block of code, if some internal
		// structs don't exist
		func() {
			var exists bool

			svcDesc, exists = currentSettings.Services[svcName]
			if !exists {
				checkerExists = false
				l.DebugContext(ctx, "checker doesn't exist in current settings, setting rate to zero")
				return
			}

			checkerDesc, exists = svcDesc.Checkers[checkerName]
			if !exists {
				checkerExists = false
				l.DebugContext(ctx, "checker doesn't exist in current settings, setting rate to zero")
				return
			}
		}()

		checkRate := raterunner.ZeroRate
		getRate := raterunner.ZeroRate
		if checkerExists {
			checkRate = raterunner.Rate{
				Times: checkerDesc.Check.Rate.Times,
				Per:   checkerDesc.Check.Rate.Per.AsDuration(),
			}
			getRate = raterunner.Rate{
				Times: checkerDesc.Get.Rate.Times,
				Per:   checkerDesc.Get.Rate.Per.AsDuration(),
			}
		}

		checkerTaskName := fmt.Sprintf("%s__%s__check", svcName, checkerName)
		checkerTask := c.genCheckerCheckTask(checker)
		if err := c.setTaskRate(checkerTaskName, checkerTask, checkRate); err != nil {
			errlist = multierror.Append(
				errlist,
				errors.Wrapf(err, "cannot set rate on check %q", checkerID),
			)
			continue
		}

		getterTaskName := fmt.Sprintf("%s__%s__get", svcName, checkerName)
		getterTask := c.genCheckerGetTask(checker)
		if err := c.setTaskRate(getterTaskName, getterTask, getRate); err != nil {
			errlist = multierror.Append(
				errlist,
				errors.Wrapf(err, "cannot set rate on get %q", checkerID),
			)
			continue
		}
	}

	for sploitID, sploit := range c.registeredSploits {
		l := c.l.With(slog.Any("sploit_id", sploitID))

		svcName, sploitName := sploitID.Service, sploitID.Name
		existingSploit := true

		var svcDesc *checkersettings.ServiceSettings
		var sploitDesc *checkersettings.SploitSettings
		func() {
			var exists bool

			svcDesc, exists = currentSettings.Services[svcName]
			if !exists {
				existingSploit = false
				l.DebugContext(ctx, "sploit doesn't exist in current settings, setting rate to zero")
				return
			}

			sploitDesc, exists = svcDesc.Sploits[sploitName]
			if !exists {
				existingSploit = false
				l.DebugContext(ctx, "sploit doesn't exist in current settings, setting rate to zero")
				return
			}
		}()

		sploitRate := raterunner.ZeroRate
		if existingSploit {
			sploitRate = raterunner.Rate{
				Times: sploitDesc.Rate.Times,
				Per:   sploitDesc.Rate.Per.AsDuration(),
			}
		}

		sploitTaskName := fmt.Sprintf("%s__%s__sploit", svcName, sploitName)
		sploitTask := c.genSploitRunAttackTask(sploit)
		if err := c.setTaskRate(sploitTaskName, sploitTask, sploitRate); err != nil {
			errlist = multierror.Append(
				errlist,
				errors.Wrap(err, "cannot set task rate for sploit"),
			)
			continue
		}
	}

	return errlist.ErrorOrNil()
}

func (c *Controller) setTaskRate(taskName string, task raterunner.TaskFunc, rate raterunner.Rate) error {
	if !c.rr.TaskRegistered(taskName) {
		if err := c.rr.RegisterTask(taskName, task); err != nil {
			return errors.Wrapf(err, "cannot register task %q to rate runner", taskName)
		}
	}

	// task is registered now
	err := c.rr.SetTaskRate(taskName, rate)
	if err != nil {
		return errors.Wrapf(err, "cannot set task rate in rate runner for task %q", taskName)
	}

	return nil
}

func (c *Controller) providerForService(id string) hazycheck.Connector {
	p := func() hazycheck.Connector {
		c.providersMu.RLock()
		defer c.providersMu.RUnlock()

		return c.cachedProviders[id]
	}()

	if p != nil {
		// provider found, return it
		return p
	}

	// provider not found, create it
	c.providersMu.Lock()
	defer c.providersMu.Unlock()

	if p, exists := c.cachedProviders[id]; exists {
		return p
	}

	p = hazycheck.NewConnector(id)
	c.cachedProviders[id] = p
	return p
}

func (c *Controller) waitFor(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
