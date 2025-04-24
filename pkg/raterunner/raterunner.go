package raterunner

import (
	"context"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/HazyCorp/govnilo/common/hzlog"
	"github.com/HazyCorp/govnilo/pkg/ratelimit"
	"github.com/HazyCorp/govnilo/taskrunner"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type TaskFunc func(ctx context.Context) error

func (f TaskFunc) Run(ctx context.Context) error {
	return f(ctx)
}

type Rate struct {
	Times uint64
	Per   time.Duration
}

var ZeroRate = Rate{Times: 0, Per: time.Second}

type taskSpec struct {
	l  *slog.Logger
	mu sync.Mutex

	f    TaskFunc
	name string

	avgCounter       *AvgCounter
	targetRate       Rate
	currentInstances uint64
	rateLimitter     *ratelimit.Limiter
}

func (t *taskSpec) neededInstances() uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	avg, err := t.avgCounter.GetAvg()
	if err != nil {
		t.l.Info(
			"cannot correct instances of task: cannot get average of it's running time, using a default value",
			slog.Duration("value", time.Second),
		)
		avg = float64(time.Second)
	}

	avgDuration := time.Duration(int64(avg))
	t.l.Debug(
		"task running stats",
		slog.Duration("avg_duration", avgDuration),
		slog.Uint64("needs", t.targetRate.Times),
		slog.Duration("per", t.targetRate.Per),
	)
	instances := uint64(
		float64(t.targetRate.Times) * float64(avgDuration) / float64(t.targetRate.Per),
	)

	// rate limitter will stop extra calls
	instances *= 2

	return instances
}

type RateRunner struct {
	mu sync.Mutex

	l *slog.Logger

	tr    *taskrunner.TaskRunner
	tasks map[string]*taskSpec
}

func New(l *slog.Logger) *RateRunner {
	return &RateRunner{
		l: l.With(slog.String("component", "rate-runner")),

		tr:    taskrunner.NewTaskRunner(l),
		tasks: make(map[string]*taskSpec),
	}
}

func (r *RateRunner) Run(ctx context.Context) (err error) {
	defer func() {
		r.l.Info("ending rateRunner.Run routine")

		tctx, tcancel := context.WithTimeout(context.Background(), time.Second)
		defer tcancel()

		if cleanupErr := r.cleanup(tctx); cleanupErr != nil {
			err = cleanupErr
		}
	}()

	for {
		if err := r.waitFor(ctx, time.Second); err != nil {
			return err
		}

		copied := make(map[string]*taskSpec)
		func() {
			r.mu.Lock()
			defer r.mu.Unlock()

			for k, v := range r.tasks {
				copied[k] = v
			}
		}()

		for taskName, task := range copied {
			l := r.l.With(slog.String("task_name", taskName))
			l.Debug("checking task if it needs to adjust amount of replicas")

			newInstances := task.neededInstances()
			l.Debug(
				"task calculated value",
				slog.Uint64("current_instances", task.currentInstances),
				slog.Uint64("calculated_instances", newInstances),
			)

			func() {
				task.mu.Lock()
				defer task.mu.Unlock()

				if newInstances == task.currentInstances {
					l.Debug("don't need to update amount of instances")
					return
				}

				if err := r.tr.UpdateTaskInstances(taskName, int(newInstances)); err != nil {
					l.Warn("cannot update amount of instances", hzlog.Error(err))
					return
				}

				task.currentInstances = newInstances
			}()
		}
	}
}

func (r *RateRunner) cleanup(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errlist *multierror.Error
	for name, t := range r.tasks {
		r.l.Debug("stopping the rate limitter", slog.String("name", name))

		if err := t.rateLimitter.Stop(ctx); err != nil {
			errlist = multierror.Append(errlist, err)
		}
	}

	err := errlist.ErrorOrNil()
	if err != nil {
		return errors.Wrap(err, "cannot cleanup raterunner")
	}

	return nil
}

func (r *RateRunner) TaskRegistered(taskName string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.tasks[taskName]
	return exists
}

func (r *RateRunner) RegisterTask(taskName string, f TaskFunc) error {
	r.l.Debug("registering task in rate runner", slog.String("task_name", taskName))

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[taskName]; exists {
		return errors.Errorf("cannot register task, task already registered")
	}

	spec := &taskSpec{
		l:          r.l.With(slog.String("task_name", taskName)),
		f:          f,
		name:       taskName,
		avgCounter: NewAvgCounter(AvgCounterSpec{WindowSize: DefaultWindowSize}),
		targetRate: Rate{Times: 0, Per: time.Second},
		rateLimitter: ratelimit.New(
			ratelimit.Spec{Times: 0, Per: time.Second},
			ratelimit.WithLogger(r.l.With(slog.String("task_name", taskName))),
		),
	}

	r.tasks[taskName] = spec

	toRegister := r.prepareTask(spec)
	if err := r.tr.RegisterTask(taskName, toRegister); err != nil {
		return errors.Wrap(err, "cannot register task to task runner")
	}

	return nil
}

func (r *RateRunner) SetTaskRate(taskName string, rate Rate) error {
	if rate.Per == 0 {
		return errors.Errorf("rate.Per must be nonzero")
	}

	var t *taskSpec
	var exists bool
	var err error
	func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		t, exists = r.tasks[taskName]
		if !exists {
			err = errors.Errorf("task %s not registered to rate runner", taskName)
			return
		}
	}()

	if err != nil {
		return err
	}

	err = t.rateLimitter.SetSpec(ratelimit.Spec{
		Times: rate.Times,
		Per:   rate.Per,
	})
	if err != nil {
		return errors.Wrap(err, "cannot set task rate")
	}

	func() {
		t.mu.Lock()
		defer t.mu.Unlock()

		t.targetRate = rate
	}()

	return nil
}

func (r *RateRunner) prepareTask(t *taskSpec) TaskFunc {
	return func(ctx context.Context) error {
		// to distribute load more uniformly let's sleep random amount of time in the beginning (jitter)
		// TODO: remove: ratelimitter should shape the load
		toSleep := time.Duration(uint64(rand.Float64() * float64(t.targetRate.Per)))
		if err := r.waitFor(ctx, toSleep); err != nil {
			return err
		}

		for {
			if err := t.rateLimitter.Acquire(ctx); err != nil {
				return err
			}

			start := time.Now()
			if err := t.f(ctx); err != nil {
				r.l.Warn("task failed in rate runner, restarting", slog.String("task_name", t.name), hzlog.Error(err))
				// we don't need to update avg on failed tasks
				continue
			}

			duration := time.Since(start)
			t.avgCounter.Append(uint64(duration))
		}
	}
}

func (r *RateRunner) waitFor(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
