package taskrunner

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/HazyCorp/govnilo/common/hzlog"
	"github.com/HazyCorp/govnilo/hazyerr"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type Task interface {
	Run(ctx context.Context) error
}

type taskInstance struct {
	ctx     context.Context
	cancel  context.CancelFunc
	stopped atomic.Bool
	errChan chan error
}

func (t *taskInstance) stop(ctx context.Context) error {
	slog.Info("stopping task instance")

	t.stopped.Store(true)
	t.cancel()

	// if err is ready, but context is already cancelled, return error from run
	select {
	case <-t.errChan:
		return nil
	default:
		// pass
	}

	select {
	case <-t.errChan:
		return nil

	case <-ctx.Done():
		slog.Info("context done received while awaiting the task instance stop")
		return ctx.Err()
	}
}

type taskState struct {
	mu        sync.Mutex
	instances []*taskInstance
}

type TaskRunner struct {
	mu sync.Mutex

	taskIDToState map[string]*taskState
	taskIDToTask  map[string]Task

	l           *slog.Logger
	baseContext context.Context
}

func NewTaskRunner(l *slog.Logger) *TaskRunner {
	return &TaskRunner{
		l:           l.With(slog.String("component", "task-runner")),
		baseContext: context.TODO(),

		taskIDToState: make(map[string]*taskState),
		taskIDToTask:  make(map[string]Task),
	}
}

func (r *TaskRunner) Start(ctx context.Context) error {
	return nil
}

func (r *TaskRunner) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errlist *multierror.Error
	for taskName, state := range r.taskIDToState {
		l := r.l.With(slog.String("task_name", taskName))

		l.Info("trying to stop task")

		func() {
			state.mu.Lock()
			defer state.mu.Unlock()

			for idx, instance := range state.instances {
				l := l.With(
					slog.Int("instance", idx+1),
					slog.Int("total_instances", len(state.instances)),
				)

				l.Info("trying to stop task instance")

				tCtx, tCancel := context.WithTimeout(ctx, time.Second)
				err := instance.stop(tCtx)
				tCancel()

				if err != nil {
					l.Warn("stopping of task instance failed")
					errlist = multierror.Append(errlist, err)
					continue
				}

				l.Info("successfully stopped instsance!")
			}
		}()
	}

	err := errlist.ErrorOrNil()
	if err != nil {
		return errors.Wrap(err, "cannot stop all running tasks")
	}

	return nil
}

func (r *TaskRunner) RegisterTask(name string, task Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, exists := r.taskIDToTask[name]
	if exists {
		return errors.Wrapf(
			hazyerr.ErrAlreadyExists,
			"cannot register task %s, task with this name already registered",
			name,
		)
	}

	r.taskIDToTask[name] = task
	r.taskIDToState[name] = &taskState{instances: nil}

	return nil
}

func (r *TaskRunner) UpdateTaskInstances(taskName string, instances int) error {
	l := r.l.With(
		slog.Int("target_instances", instances),
		slog.String("task_name", taskName),
	)

	l.Debug("trying to update amount of instances")

	var state *taskState
	var task Task
	var exists bool
	func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		task, exists = r.taskIDToTask[taskName]
		if !exists {
			return
		}

		state, exists = r.taskIDToState[taskName]
		if !exists {
			return
		}
	}()

	if !exists {
		return errors.Wrapf(
			hazyerr.ErrNotFound,
			"cannot update task instances of task %s, task not registered",
			taskName,
		)
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	if len(state.instances) == instances {
		l.Info("updating task instances of task is actually noop, skip")
		return nil
	}

	if len(state.instances) < instances {
		toAdd := instances - len(state.instances)

		l.Info("adding additional tasks", slog.Int("to_add", toAdd))
		for i := 0; i < toAdd; i++ {
			ctx, cancel := context.WithCancel(r.baseContext)

			instance := &taskInstance{
				ctx:     ctx,
				cancel:  cancel,
				stopped: atomic.Bool{},
				errChan: make(chan error, 1),
			}

			go func() {
				for {
					err := task.Run(ctx)
					if !instance.stopped.Load() {
						r.l.Debug(
							"task ended it's job with err, but task not stopped, restarting",
							hzlog.Error(err),
							slog.String("task_name", taskName),
						)
						continue
					}

					r.l.Debug(
						"task ended it's job with err, and task is stopped, stopping the routine",
						hzlog.Error(err),
						slog.String("task_name", taskName),
					)

					instance.errChan <- err
					return
				}
			}()

			state.instances = append(state.instances, instance)
		}

		l.Info("started new tasks")

		return nil
	}

	// need to stop some instances of provided taskName
	toStop := len(state.instances) - instances
	l.Info("need to stop tasks", slog.Int("to_stop", toStop))

	var errlist *multierror.Error
	for i := 0; i < toStop; i++ {
		taskInstance := state.instances[len(state.instances)-1]
		// TODO: use timeouts from some config
		tCtx, tCancel := context.WithTimeout(r.baseContext, time.Second)
		err := taskInstance.stop(tCtx)
		tCancel()

		if err != nil {
			errlist = multierror.Append(errlist, err)
		}

		state.instances = state.instances[:len(state.instances)-1]
	}

	err := errlist.ErrorOrNil()
	if err != nil {
		return errors.Wrapf(
			err,
			"cannot stop %d/%d task instances of %s",
			len(errlist.Errors),
			toStop,
			taskName,
		)
	}

	return nil
}
