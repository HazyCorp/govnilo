package raterunner

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/HazyCorp/govnilo/pkg/ratelimit"
	"github.com/HazyCorp/govnilo/internal/taskrunner"
)

type TaskFunc func(ctx context.Context) error

func (f TaskFunc) Run(ctx context.Context) error {
	return f(ctx)
}

type Rate struct {
	Times uint64
	Per   time.Duration
}

type taskSpec struct {
	l  *zap.Logger
	mu sync.Mutex

	f    TaskFunc
	name string

	avgCounter       *AvgCounter
	targetRate       Rate
	currentInstances uint64
	rateLimitter     *ratelimit.RateLimitter
}

func (t *taskSpec) neededInstances() uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	avg, err := t.avgCounter.GetAvg()
	if err != nil {
		t.l.Sugar().
			Infof("cannot correct instances of task %s: cannot get average of it's running time, using %s as a default", t.name, time.Second)
		avg = float64(time.Second)
	}

	avgDuration := time.Duration(int64(avg))
	t.l.Sugar().
		Debugf("average time of running %q is %s, need %d times per %s", t.name, avgDuration, t.targetRate.Times, t.targetRate.Per)

	instances := uint64(
		float64(t.targetRate.Times) * float64(avgDuration) / float64(t.targetRate.Per),
	)
	// rate limitter will stop extra calls
	instances += 1

	return instances
}

type RateRunner struct {
	mu sync.Mutex

	l *zap.Logger

	tr    *taskrunner.TaskRunner
	tasks map[string]*taskSpec
}

func New(l *zap.Logger) *RateRunner {
	return &RateRunner{
		l: l,

		tr:    taskrunner.NewTaskRunner(l),
		tasks: make(map[string]*taskSpec),
	}
}

func (r *RateRunner) Run(ctx context.Context) (err error) {
	defer func() {
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

		copied := make(map[string]*taskSpec, 0)
		func() {
			r.mu.Lock()
			defer r.mu.Unlock()

			for k, v := range r.tasks {
				copied[k] = v
			}
		}()

		for taskName, task := range copied {
			r.l.Sugar().
				Debugf("checking task %s if it needs to adjust amount of replicas", taskName)

			newInstances := task.neededInstances()
			r.l.Sugar().
				Debugf("%s task has %d instances running, new calculated value: %d", taskName, task.currentInstances, newInstances)

			func() {
				task.mu.Lock()
				defer task.mu.Unlock()

				if newInstances == task.currentInstances {
					r.l.Debug("don't need to update amount of instances")
					return
				}

				r.l.Sugar().
					Debugf("updating instances of task %s from %d to %d", taskName, task.currentInstances, newInstances)

				if err := r.tr.UpdateTaskInstances(taskName, int(newInstances)); err != nil {
					r.l.Sugar().Warnf("cannot update amount of instances of %s", taskName)
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
		r.l.Sugar().Debugf("stopping %q rate limitter", name)

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
	r.l.Sugar().Debugf("registering %s task in rate runner", taskName)

	r.mu.Lock()
	defer r.mu.Unlock()

	spec := &taskSpec{
		l:          r.l.With(zap.String("task_name", taskName)),
		f:          f,
		name:       taskName,
		avgCounter: NewAvgCounter(AvgCounterSpec{WindowSize: DefaultWindowSize}),
		targetRate: Rate{Times: 0, Per: time.Second},
		rateLimitter: ratelimit.New(
			r.l.With(zap.String("task_name", taskName)),
			// zap.NewNop(),
			ratelimit.RateLimitterSpec{Times: 0, Per: time.Second},
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

	r.mu.Lock()
	defer r.mu.Unlock()

	t, exists := r.tasks[taskName]
	if !exists {
		return errors.Errorf("task %s not registered to rate runner", taskName)
	}

	err := t.rateLimitter.SetSpec(ratelimit.RateLimitterSpec{
		Times: rate.Times,
		Per:   rate.Per,
	})
	if err != nil {
		return errors.Wrap(err, "cannot set task rate")
	}

	t.mu.Lock()
	t.targetRate = rate
	t.mu.Unlock()

	return nil
}

func (r *RateRunner) prepareTask(t *taskSpec) TaskFunc {
	return func(ctx context.Context) error {
		// to distribute load more uniformly let's sleep random amount of time in the beginning (jitter)
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
				r.l.
					With(zap.Error(err)).
					Sugar().
					Warnf("task %s failed in rate runner, retrying", t.name)
			}

			duration := time.Since(start)
			t.avgCounter.Append(uint64(duration))
		}
	}
}

func (r *RateRunner) waitFor(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)

	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		t.Stop()
		return ctx.Err()
	}
}
