package checkerctrl

import (
	"context"
	"fmt"
	"github.com/HazyCorp/govnilo/common/checkersettings"
	"github.com/HazyCorp/govnilo/common/hzlog"
	hazycheck2 "github.com/HazyCorp/govnilo/hazycheck"
	"github.com/HazyCorp/govnilo/raterunner"
	"log/slog"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

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
	registeredCheckers map[hazycheck2.CheckerID]hazycheck2.Checker
	registeredSploits  map[hazycheck2.SploitID]hazycheck2.Sploit
	storage            ControllerStorage
	settingsProvider   SettingsProvider
	conf               Config
	strategy           SaveStrategy

	runCtx     context.Context
	runCancel  context.CancelFunc
	runErrChan chan error

	checkerMetricsMu sync.RWMutex
	checkerMetrics   map[hazycheck2.CheckerID]*checkerMetrics

	sploitMetricsMu sync.RWMutex
	sploitMetrics   map[hazycheck2.SploitID]*sploitMetrics

	providersMu     sync.RWMutex
	cachedProviders map[string]hazycheck2.Connector

	currentSettings atomic.Pointer[checkersettings.Settings]

	rr *raterunner.RateRunner
}

type ControllerIn struct {
	fx.In

	Logger           *slog.Logger
	Checkers         []hazycheck2.Checker `group:"checkers"`
	Sploits          []hazycheck2.Sploit  `group:"sploits"`
	Storage          ControllerStorage
	SettingsProvider SettingsProvider
	Config           Config
	Strategy         SaveStrategy
}

func New(in ControllerIn) *Controller {
	idToChecker := make(map[hazycheck2.CheckerID]hazycheck2.Checker, len(in.Checkers))
	for _, check := range in.Checkers {
		idToChecker[check.CheckerID()] = check
	}

	idToSploit := make(map[hazycheck2.SploitID]hazycheck2.Sploit, len(in.Sploits))
	for _, sploit := range in.Sploits {
		idToSploit[sploit.SploitID()] = sploit
	}

	return &Controller{
		l:                  in.Logger,
		registeredCheckers: idToChecker,
		registeredSploits:  idToSploit,
		storage:            in.Storage,
		rr:                 raterunner.New(in.Logger),
		conf:               in.Config,
		runErrChan:         make(chan error, 1),
		strategy:           in.Strategy,
		settingsProvider:   in.SettingsProvider,
		checkerMetrics:     make(map[hazycheck2.CheckerID]*checkerMetrics),
		sploitMetrics:      make(map[hazycheck2.SploitID]*sploitMetrics),

		cachedProviders: make(map[string]hazycheck2.Connector),
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
	sploit hazycheck2.Sploit,
) raterunner.TaskFunc {
	sploitID := sploit.SploitID()
	m := c.sploitMetricsFor(sploitID)

	return func(ctx context.Context) error {
		currentSettings := c.currentSettings.Load()
		serviceSettings := currentSettings.Services[sploitID.Service]
		if serviceSettings == nil {
			return errors.Errorf("cannot find service %s in current settings", sploitID.Service)
		}

		checkerSettings := serviceSettings.Checkers[sploitID.Name]
		if checkerSettings == nil {
			return errors.Errorf("cannot find sploit %s in %s service settings", sploitID.Name, sploitID.Service)
		}

		start := time.Now()

		// TODO: optimize, build new target only on new config set
		target := fmt.Sprintf("%s:%d", c.conf.TargetHost, serviceSettings.TargetPort)
		err := sploit.RunAttack(ctx, target)
		if err != nil {
			m.FailCounter.Inc()
			m.FailDuration.UpdateDuration(start)

			return errors.Wrap(err, "sploit.RunAttack unexpectedly errored")
		}

		m.SuccessCounter.Inc()
		m.SuccessDuration.UpdateDuration(start)

		// don't need to save anything after sploit running
		return nil
	}
}

func (c *Controller) genCheckerCheckTask(
	checker hazycheck2.Checker,
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

		// TODO: optimize, build new target only on new config set
		target := fmt.Sprintf("%s:%d", c.conf.TargetHost, serviceSettings.TargetPort)
		data, err := checker.Check(ctx, target)

		success := true
		if err != nil {
			l.Info(
				"checker.Check of errored",
				slog.Any("checker", checker.CheckerID()),
				hzlog.Error(err),
			)

			// if checker.Check fails -- it's OK, it's expected behaviour, so, we don't need to fail this task
			success = false
		}

		if success {
			l.Debug("checker run succeed", slog.Any("checker_id", checker.CheckerID()))
			m.SuccessCheckCounter.Inc()
			m.SuccessCheckDuration.UpdateDuration(start)
			m.SuccessCheckPoints.Add(int(checkerSettings.Check.SuccessPoints))
		} else {
			m.FailCheckCounter.Inc()
			m.FailCheckDuration.UpdateDuration(start)
			m.FailCheckPenalty.Add(int(checkerSettings.Check.FailPenalty))
		}

		// but, errors of saving state are unexpected, need to mark task as failed

		l.Debug("saving the data", slog.String("data", string(data)))
		if err := c.saveCheckerData(ctx, checker.CheckerID(), data); err != nil {
			return errors.Wrap(err, "cannot save check data to storage")
		}

		return nil
	}
}

func (c *Controller) genCheckerGetTask(
	checker hazycheck2.Checker,
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

		idx := rand.Intn(len(pool))
		data := pool[idx]

		start := time.Now()
		success := true
		// TODO: optimize, build new target only on new config set
		target := fmt.Sprintf("%s:%d", c.conf.TargetHost, serviceSettings.TargetPort)
		err = checker.Get(ctx, target, data)
		if err != nil {
			// if checker.Get fails -- it's OK, it's expected behaviour, so, we jkon't need to fail this task
			success = false
			l.Debug("checker.Get of failed with message", hzlog.Error(err))
		}

		if success {
			l.Debug("checker.Get succeed")
			m.SuccessGetCounter.Inc()
			m.SuccessGetDuration.UpdateDuration(start)
			m.SuccessGetPoints.Add(int(checkerSettings.Get.SuccessPoints))
		} else {
			m.FailGetCounter.Inc()
			m.FailGetDuration.UpdateDuration(start)
			m.FailGetPenalty.Add(int(checkerSettings.Get.FailPenalty))
		}

		return nil
	}
}

func (c *Controller) saveCheckerData(
	ctx context.Context,
	checkerID hazycheck2.CheckerID,
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

	toDelete := c.strategy.NeedDelete(uint64(len(pool)))
	for _, idx := range toDelete {
		if _, err := c.storage.RemoveDataFromPool(ctx, checkerID, idx); err != nil {
			return errors.Wrap(err, "cannot delete stale data from pool")
		}
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

	c.currentSettings.Store(currentSettings)

	// TODO: handle removal of missing service tasks
	var errlist *multierror.Error
	for svcName, svc := range currentSettings.Services {
		for checkerName, checkerState := range svc.Checkers {
			checkerID := hazycheck2.CheckerID{
				Service: svcName,
				Name:    checkerName,
			}

			checker, exists := c.registeredCheckers[checkerID]
			if !exists {
				errlist = multierror.Append(
					errlist,
					errors.Errorf(
						"checker %q from saved state doesn't exist",
						svcName,
					),
				)
				continue
			}

			checkerTaskName := fmt.Sprintf("%s__%s__%s", svcName, checkerName, "check")
			checkerTask := c.genCheckerCheckTask(checker)
			if err := c.setTaskRate(checkerTaskName, checkerTask, checkerState.Check.Rate); err != nil {
				errlist = multierror.Append(
					errlist,
					errors.Wrapf(err, "cannot set rate on check %q", checker.CheckerID()),
				)
				continue
			}

			getterTaskName := fmt.Sprintf("%s__%s__%s", svcName, checkerName, "get")
			getterTask := c.genCheckerGetTask(checker)
			if err := c.setTaskRate(getterTaskName, getterTask, checkerState.Get.Rate); err != nil {
				errlist = multierror.Append(
					errlist,
					errors.Wrapf(err, "cannot set rate on get %q", checker.CheckerID()),
				)
				continue
			}
		}

		for sploitName, settings := range svc.Sploits {
			sploitID := hazycheck2.SploitID{Service: svcName, Name: sploitName}
			sploit, exists := c.registeredSploits[sploitID]
			if !exists {
				errlist = multierror.Append(
					errlist,
					errors.Errorf(
						"sploit %s/%s not registered to checker, but it is present in the config",
						svcName, sploitName,
					),
				)

				continue
			}

			sploitTaskName := fmt.Sprintf("%s__%s__sploit", svcName, sploitName)
			sploitTask := c.genSploitRunAttackTask(sploit)
			if err := c.setTaskRate(sploitTaskName, sploitTask, settings.Rate); err != nil {
				errlist = multierror.Append(
					errlist,
					errors.Wrap(err, "cannot set task rate for sploit"),
				)
				continue
			}
		}
	}

	return errlist.ErrorOrNil()
}

func (c *Controller) setTaskRate(taskName string, task raterunner.TaskFunc, rate checkersettings.Rate) error {
	if !c.rr.TaskRegistered(taskName) {
		if err := c.rr.RegisterTask(taskName, task); err != nil {
			return errors.Wrapf(err, "cannot register task %q to rate runner", taskName)
		}
	}

	// task is registered now
	err := c.rr.SetTaskRate(taskName, raterunner.Rate{
		Times: rate.Times,
		Per:   time.Duration(rate.Per),
	})
	if err != nil {
		return errors.Wrapf(err, "cannot set task rate in rate runner for task %q", taskName)
	}

	return nil
}

func (c *Controller) providerForService(id string) hazycheck2.Connector {
	p := func() hazycheck2.Connector {
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

	p = hazycheck2.NewConnector(id)
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
