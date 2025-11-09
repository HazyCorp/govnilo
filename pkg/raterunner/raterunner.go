package raterunner

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HazyCorp/govnilo/internal/taskrunner"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"
	"github.com/HazyCorp/govnilo/pkg/ratelimit"
	"github.com/VictoriaMetrics/metrics"

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

type RunOptions struct {
	Rate          Rate
	MaxGoroutines int
}

type TaskHandle struct {
	spec *taskSpec
}

func (h *TaskHandle) SetOptions(options RunOptions) error {
	if options.Rate.Per == 0 {
		return errors.Errorf("rate.Per must be nonzero")
	}

	// Update rate limiter spec first to avoid races between limiter and run options.
	if err := h.spec.rateLimitter.SetSpec(ratelimit.Spec{
		Times: options.Rate.Times,
		Per:   options.Rate.Per,
	}); err != nil {
		return errors.Wrapf(err, "cannot set task rate for %q", h.spec.id)
	}

	h.spec.mu.Lock()
	defer h.spec.mu.Unlock()

	h.spec.runOptions = options
	return nil
}

type taskSpec struct {
	l  *slog.Logger
	mu sync.Mutex

	f  TaskFunc
	id string

	stat             Stat
	runOptions       RunOptions
	currentInstances uint64
	rateLimitter     *ratelimit.Limiter
}

func (t *taskSpec) neededInstances() uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	stat, err := t.stat.GetStat()
	if err != nil {
		t.l.Debug(
			"cannot correct instances of task: cannot get stat of it's running time, using a default value",
			slog.Duration("value", time.Second),
		)
		stat = float64(time.Second)
	}

	statisticalDuration := time.Duration(int64(stat))
	t.l.Debug(
		"task running stats",
		slog.Duration("avg_duration", statisticalDuration),
		slog.Uint64("needs", t.runOptions.Rate.Times),
		slog.Duration("per", t.runOptions.Rate.Per),
		slog.Int("max_goroutines", t.runOptions.MaxGoroutines),
	)
	instances := uint64(
		float64(t.runOptions.Rate.Times) * float64(statisticalDuration) / float64(t.runOptions.Rate.Per),
	)

	// rate limitter will stop extra calls
	instances = uint64(float64(instances)*1.2) + 1

	// Return instances WITHOUT applying max_goroutines limit
	return instances
}

type RateRunner struct {
	mu sync.Mutex

	l *slog.Logger

	tr    *taskrunner.TaskRunner
	tasks map[string]*taskSpec

	idCounter atomic.Uint64
}

func New(l *slog.Logger) *RateRunner {
	return &RateRunner{
		l: l.With(slog.String("component", "infra:rate-runner")),

		tr:        taskrunner.NewTaskRunner(l),
		tasks:     make(map[string]*taskSpec),
		idCounter: atomic.Uint64{},
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

		for taskID, task := range copied {
			l := r.l.With(slog.String("task_id", taskID))
			l.Debug("checking task if it needs to adjust amount of replicas")

			// Get needed instances without limit
			neededInstances := task.neededInstances()

			metrics.GetOrCreateGauge(
				fmt.Sprintf("checker_raterunner_needed_instances{task_id=%q}", taskID),
				nil,
			).Set(float64(neededInstances))
			metrics.GetOrCreateGauge(
				fmt.Sprintf("checker_raterunner_max_goroutines{task_id=%q}", taskID),
				nil,
			).Set(float64(task.runOptions.MaxGoroutines))
			metrics.GetOrCreateGauge(
				fmt.Sprintf("checker_raterunner_current_instances{task_id=%q}", taskID),
				nil,
			).Set(float64(task.currentInstances))

			// Check if we need to warn about goroutine limit and apply max_goroutines limit if needed
			if task.runOptions.MaxGoroutines > 0 && neededInstances > uint64(task.runOptions.MaxGoroutines) {
				neededInstances = uint64(task.runOptions.MaxGoroutines)
				l.Warn(
					"needed instances exceed max_goroutines limit",
					slog.Uint64("needed_instances", neededInstances),
					slog.Int("max_goroutines", task.runOptions.MaxGoroutines),
					slog.Uint64("limited_to", uint64(task.runOptions.MaxGoroutines)),
				)
			}

			l.Debug(
				"task calculated value",
				slog.Uint64("current_instances", task.currentInstances),
				slog.Uint64("needed_instances", neededInstances),
			)

			func() {
				task.mu.Lock()
				defer task.mu.Unlock()

				if neededInstances == task.currentInstances {
					l.Debug("don't need to update amount of instances")
					return
				}

				if err := r.tr.UpdateTaskInstances(taskID, int(neededInstances)); err != nil {
					l.Warn("cannot update amount of instances", hzlog.Error(err))
					return
				}

				task.currentInstances = neededInstances
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

func (r *RateRunner) RegisterTask(f TaskFunc) (*TaskHandle, error) {
	options := RunOptions{
		Rate:          Rate{Times: 0, Per: time.Second},
		MaxGoroutines: 0,
	}
	taskID := fmt.Sprintf("task-%d", r.idCounter.Add(1))

	r.l.Debug("registering task in rate runner", slog.String("task_id", taskID), slog.Int("max_goroutines", options.MaxGoroutines))

	r.mu.Lock()
	defer r.mu.Unlock()

	spec := &taskSpec{
		l:          r.l.With(slog.String("task_id", taskID)),
		f:          f,
		id:         taskID,
		stat:       NewPercentile(0.95, time.Minute),
		runOptions: options,
		rateLimitter: ratelimit.New(
			ratelimit.Spec{Times: options.Rate.Times, Per: options.Rate.Per},
			ratelimit.WithLogger(r.l.With(slog.String("task_id", taskID))),
		),
	}

	r.tasks[taskID] = spec

	toRegister := r.prepareTask(spec)
	if err := r.tr.RegisterTask(taskID, toRegister); err != nil {
		delete(r.tasks, taskID)
		return nil, errors.Wrap(err, "cannot register task to task runner")
	}

	handle := &TaskHandle{
		spec: spec,
	}

	return handle, nil
}

func (r *RateRunner) prepareTask(t *taskSpec) TaskFunc {
	return func(ctx context.Context) error {
		// to distribute load more uniformly let's sleep random amount of time in the beginning (jitter)
		// TODO: remove: ratelimitter should shape the load
		toSleep := time.Duration(uint64(rand.Float64() * float64(t.runOptions.Rate.Per)))
		if err := r.waitFor(ctx, toSleep); err != nil {
			return err
		}

		for {
			if err := t.rateLimitter.Acquire(ctx); err != nil {
				return err
			}

			start := time.Now()
			if err := t.f(ctx); err != nil {
				r.l.Warn("task failed in rate runner, restarting", slog.String("task_id", t.id), hzlog.Error(err))
				// we don't need to update avg on failed tasks
				continue
			}

			duration := time.Since(start)
			t.stat.Append(uint64(duration))
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
