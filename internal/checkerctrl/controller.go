package checkerctrl

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/internal/taskrunner"
	"github.com/HazyCorp/govnilo/pkg/common/checkersettings"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"
	"github.com/HazyCorp/govnilo/pkg/raterunner"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"go.uber.org/fx"
	"golang.org/x/sync/errgroup"
)

type Config struct {
	SyncInterval time.Duration `json:"sync_interval" yaml:"sync_interval"`
}

type Controller struct {
	l                  *slog.Logger
	registeredCheckers map[hazycheck.CheckerID]hazycheck.Checker
	checkerHandles     map[hazycheck.CheckerID]*raterunner.TaskHandle
	settingsProvider   SettingsProvider
	conf               Config

	runCtx     context.Context
	runCancel  context.CancelFunc
	runErrChan chan error

	checkerMetrics map[hazycheck.CheckerID]*checkerMetrics

	currentSettings atomic.Pointer[checkersettings.Settings]

	rr *raterunner.RateRunner
}

type ControllerIn struct {
	fx.In

	Logger           *slog.Logger
	Checkers         []hazycheck.Checker `group:"checkers"`
	SettingsProvider SettingsProvider
	Config           Config
}

func New(in ControllerIn) (*Controller, error) {
	c := &Controller{
		l:                  in.Logger.With(slog.String("component", "infra:checker-controller")),
		registeredCheckers: make(map[hazycheck.CheckerID]hazycheck.Checker, len(in.Checkers)),
		checkerHandles:     make(map[hazycheck.CheckerID]*raterunner.TaskHandle, len(in.Checkers)),
		checkerMetrics:     make(map[hazycheck.CheckerID]*checkerMetrics, len(in.Checkers)),
		rr:                 raterunner.New(in.Logger),
		conf:               in.Config,
		runErrChan:         make(chan error, 1),
		settingsProvider:   in.SettingsProvider,
	}

	for _, checker := range in.Checkers {
		c.registeredCheckers[checker.CheckerID()] = checker

		handle, err := c.rr.RegisterTask(c.genCheckerCheckTask(checker))
		if err != nil {
			return nil, errors.Wrap(err, "cannot register checker check task")
		}
		c.checkerHandles[checker.CheckerID()] = handle

		c.registerMetrics(checker.CheckerID())
	}

	return c, nil
}

func NewFX(in ControllerIn, lc fx.Lifecycle, sd fx.Shutdowner) (*Controller, error) {
	ctrl, err := New(in)
	if err != nil {
		return nil, errors.Wrap(err, "cannot create checker controller")
	}

	lc.Append(fx.Hook{
		OnStart: ctrl.Start,
		OnStop:  ctrl.Stop,
	})

	return ctrl, err
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
			c.l.DebugContext(ctx, "checker state syncronized")
		}
	}
}

func (c *Controller) genCheckerCheckTask(
	checker hazycheck.Checker,
) raterunner.TaskFunc {
	checkerID := checker.CheckerID()
	m := c.checkerMetricsFor(checkerID)
	l := c.l.With("checker_id", checkerID)
	// Generate trace ID for debugging (automatically adds to hzlog context)
	tracer := otel.Tracer("govnilo/checker")

	return func(ctx context.Context) error {
		ctx, span := tracer.Start(ctx, "checker.Check")
		ctx = hzlog.ContextWith(ctx, slog.Any("checker_id", checkerID))
		defer span.End()

		currentSettings := c.currentSettings.Load()
		serviceSettings := currentSettings.Services[checkerID.Service]
		if serviceSettings == nil {
			return errors.Errorf("cannot find service %s in current settings", checkerID.Service)
		}

		span.SetAttributes(attribute.String("target", serviceSettings.Target))

		l := hzlog.GetLogger(ctx, l).With(
			slog.String("target", serviceSettings.Target),
			slog.String("component", "business-infra:checker"),
		)

		checkerSettings := serviceSettings.Checkers[checkerID.Name]
		if checkerSettings == nil {
			return errors.Errorf("cannot find checker %s in %s service settings", checkerID.Name, checkerID.Service)
		}

		start := time.Now()
		var checkErr error
		cancelled := false
		// we use defer here to recover from panics
		defer func() {
			if r := recover(); r != nil {
				checkErr = hazycheck.InternalError(errors.Errorf("checker check paniced with message: %+v", checkErr))
			}

			if cancelled {
				// don't change anything
				return
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

				l.DebugContext(ctx, "checker.Check errored", hzlog.Error(checkErr))

				// if checker.Check fails -- it's OK, it's expected behaviour, so, we don't need to fail this task
				success = false
			}

			if success {
				l.DebugContext(ctx, "checker.Check run succeed")

				m.SuccessCheckCounter.Inc()
				m.SuccessCheckDuration.UpdateDuration(start)
				m.SuccessCheckPoints.Add(checkerSettings.Check.SuccessPoints)
			} else {
				l.DebugContext(ctx, "checker.Check run failed", hzlog.Error(checkErr))

				m.FailCheckCounter.Inc()
				m.FailCheckDuration.UpdateDuration(start)
				m.FailCheckPenalty.Add(checkerSettings.Check.FailPenalty)
			}
		}()

		l.DebugContext(ctx, "govnilo is running checker.Check")
		checkErr = checker.Check(ctx, serviceSettings.Target)

		if ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) {
			cause := context.Cause(ctx)
			if errors.Is(cause, taskrunner.ErrTaskCancelled) {
				l.DebugContext(ctx, "checker.Check cancelled by task runner")
				cancelled = true
			}
		}

		return nil
	}
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

	for checkerID := range c.registeredCheckers {
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

		checkOptions := raterunner.RunOptions{
			Rate: raterunner.ZeroRate,
		}
		if checkerExists {
			checkOptions = raterunner.RunOptions{
				Rate: raterunner.Rate{
					Times: checkerDesc.Check.RunOptions.Rate.Times,
					Per:   checkerDesc.Check.RunOptions.Rate.Per.AsDuration(),
				},
				MaxGoroutines: checkerDesc.Check.RunOptions.MaxGoroutines,
			}
		}

		handle, exists := c.checkerHandles[checkerID]
		if !exists {
			errlist = multierror.Append(
				errlist,
				errors.Errorf("checker handle %q not registered", checkerID),
			)
			continue
		}

		if err := handle.SetOptions(checkOptions); err != nil {
			errlist = multierror.Append(
				errlist,
				errors.Wrapf(err, "cannot set rate on check %q", checkerID),
			)
			continue
		}
	}

	return errlist.ErrorOrNil()
}

func (c *Controller) waitFor(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
