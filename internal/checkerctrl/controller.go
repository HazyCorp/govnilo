package checkerctrl

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/HazyCorp/checker/internal/raterunner"
	"github.com/HazyCorp/checker/pkg/hazycheck"
)

type Config struct {
	SyncInterval time.Duration `json:"sync_interval" yaml:"sync_interval"`
}

type Controller struct {
	l                  *zap.Logger
	registeredCheckers map[hazycheck.CheckerID]hazycheck.Checker
	registeredSploits  map[hazycheck.SploitID]hazycheck.Sploit
	storage            ControllerStorage
	conf               Config
	strategy           SaveStrategy

	runCtx     context.Context
	runCancel  context.CancelFunc
	runErrChan chan error

	checkerMetricsMu sync.RWMutex
	checkerMetrics   map[hazycheck.CheckerID]*checkerMetrics

	sploitMetricsMu sync.RWMutex
	sploitMetrics   map[hazycheck.SploitID]*sploitMetrics

	rr *raterunner.RateRunner
}

type ControllerIn struct {
	fx.In

	Logger   *zap.Logger
	Checkers []hazycheck.Checker `group:"checkers"`
	Sploits  []hazycheck.Sploit  `group:"sploits"`
	Storage  ControllerStorage
	Config   Config
	Strategy SaveStrategy
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
		l:                  in.Logger,
		registeredCheckers: idToChecker,
		registeredSploits:  idToSploit,
		storage:            in.Storage,
		rr:                 raterunner.New(in.Logger),
		conf:               in.Config,
		runErrChan:         make(chan error, 1),
		strategy:           in.Strategy,
		checkerMetrics:     make(map[hazycheck.CheckerID]*checkerMetrics),
		sploitMetrics:      make(map[hazycheck.SploitID]*sploitMetrics),
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
	if err := c.syncState(ctx); err != nil {
		return errors.Wrap(err, "cannot initially sync state")
	}

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

	for {
		if err := c.waitFor(ctx, c.conf.SyncInterval); err != nil {
			return err
		}

		if err := c.syncState(ctx); err != nil {
			c.l.Warn(
				"error occurred, while trying to sync checker controller state with storage",
				zap.Error(err),
			)
			continue
		}
	}
}

func (c *Controller) genSploitRunAttackTask(
	sploit hazycheck.Sploit,
	target string,
) raterunner.TaskFunc {
	m := c.sploitMetricsFor(sploit.SploitID())

	return func(ctx context.Context) error {
		start := time.Now()

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
	checker hazycheck.Checker,
	svc *ServiceState,
) raterunner.TaskFunc {
	m := c.checkerMetricsFor(checker.CheckerID())

	return func(ctx context.Context) error {
		start := time.Now()
		data, err := checker.Check(ctx, svc.Target)

		success := true
		if err != nil {
			c.l.
				With(zap.Error(err)).
				Sugar().
				Errorf("checker.Check of %q errored", checker.CheckerID())

			// if checker.Check fails -- it's OK, it's expected behaviour, so, we don't need to fail this task
			success = false
		}

		if success {
			c.l.Sugar().Debugf("checker %q run succeed", checker.CheckerID())
			m.SuccessCheckCounter.Inc()
			m.SuccessCheckDuration.UpdateDuration(start)
		} else {
			m.FailCheckCounter.Inc()
			m.FailCheckDuration.UpdateDuration(start)
		}

		// but, errors of saving state are unexpected, need to mark task as failed

		if _, err := c.storage.AppendCheck(ctx, checker.CheckerID(), success); err != nil {
			return errors.Wrap(err, "cannot save sla to storage")
		}

		if err := c.saveCheckerData(ctx, checker.CheckerID(), data); err != nil {
			return errors.Wrap(err, "cannot save check data to storage")
		}

		return nil
	}
}

func (c *Controller) genCheckerGetTask(
	checker hazycheck.Checker,
	svc *ServiceState,
) raterunner.TaskFunc {
	m := c.checkerMetricsFor(checker.CheckerID())

	return func(ctx context.Context) error {
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
		err = checker.Get(ctx, svc.Target, data)
		if err != nil {
			// if checker.Get fails -- it's OK, it's expected behaviour, so, we jkon't need to fail this task
			success = false
			c.l.Sugar().
				Debugf("checker.Get of %q failed with message %s", checker.CheckerID(), err)
		}

		if success {
			c.l.Sugar().Debugf("checker.Get of %q succeed", checker.CheckerID())
			m.SuccessGetCounter.Inc()
			m.SuccessGetDuration.UpdateDuration(start)
		} else {
			m.FailGetCounter.Inc()
			m.FailGetDuration.UpdateDuration(start)
		}

		// TODO: add multiplier of get fails

		// but, errors of saving state are unexpected. need to mark task as failed.
		if _, err := c.storage.AppendCheck(ctx, checker.CheckerID(), success); err != nil {
			return errors.Wrap(err, "cannot save sla to storage")
		}

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
		c.storage.AppendCheckerData(ctx, checkerID, data)
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

	currentState, err := c.storage.GetContestState(ctx)
	if err != nil {
		return errors.Wrap(
			err,
			"cannot receive routines state from storage",
		)
	}

	var errlist *multierror.Error
	for svcName, svc := range currentState.Services {
		for checkerName, checkerState := range svc.Checkers {
			checkerID := hazycheck.CheckerID{
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
			checkerTask := c.genCheckerCheckTask(checker, &svc)
			if err := c.setTaskRate(checkerTaskName, checkerTask, checkerState.Check.Rate); err != nil {
				errlist = multierror.Append(
					errlist,
					errors.Wrapf(err, "cannot set rate on check %q", checker.CheckerID()),
				)
				continue
			}

			getterTaskName := fmt.Sprintf("%s__%s__%s", svcName, checkerName, "get")
			getterTask := c.genCheckerGetTask(checker, &svc)
			if err := c.setTaskRate(getterTaskName, getterTask, checkerState.Get.Rate); err != nil {
				errlist = multierror.Append(
					errlist,
					errors.Wrapf(err, "cannot set rate on get %q", checker.CheckerID()),
				)
				continue
			}
		}

		for sploitName, state := range svc.Sploits {
			sploitID := hazycheck.SploitID{Service: svcName, Name: sploitName}
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
			sploitTask := c.genSploitRunAttackTask(sploit, svc.Target)
			if err := c.setTaskRate(sploitTaskName, sploitTask, state.Rate); err != nil {
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

func (c *Controller) setTaskRate(taskName string, task raterunner.TaskFunc, rate Rate) error {
	if !c.rr.TaskRegistered(taskName) {
		if err := c.rr.RegisterTask(taskName, task); err != nil {
			return errors.Wrapf(err, "cannot register task %q to rate runner", taskName)
		}
	}

	// task is registered now
	err := c.rr.SetTaskRate(taskName, raterunner.Rate{
		Times: rate.Times,
		Per:   rate.Per,
	})
	if err != nil {
		return errors.Wrapf(err, "cannot set task rate in rate runner for task %q", taskName)
	}

	return nil
}

func (c *Controller) waitFor(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
