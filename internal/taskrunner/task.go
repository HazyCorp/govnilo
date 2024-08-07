package taskrunner

import (
	"context"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/HazyCorp/govnilo/pkg/hazyerr"
)

type Task interface {
	Run(ctx context.Context) error
}

type taskInstance struct {
	ctx     context.Context
	cancel  context.CancelFunc
	errChan chan error
}

func (t *taskInstance) stop(ctx context.Context) error {
	t.cancel()

	// if err is ready, but context is already cancelled, return error from run
	select {
	case err := <-t.errChan:
		return err
	default:
		// pass
	}

	select {
	case err := <-t.errChan:
		return err
	case <-ctx.Done():
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

	l           *zap.Logger
	baseContext context.Context
}

func NewTaskRunner(l *zap.Logger) *TaskRunner {
	return &TaskRunner{
		l:           l,
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
		r.l.Sugar().Infof("trying to stop %s task instances", taskName)

		func() {
			state.mu.Lock()
			defer state.mu.Unlock()

			for idx, instance := range state.instances {
				r.l.Sugar().
					Infof("trying to stop %d/%d instance of %s tasks", idx+1, len(state.instances), taskName)

				tCtx, tCancel := context.WithTimeout(ctx, time.Second)
				err := instance.stop(tCtx)
				tCancel()

				if err != nil {
					r.l.Sugar().
						Infof("stopping of %d/%d instance of %s failed", idx+1, len(state.instances), taskName)
					errlist = multierror.Append(errlist, err)
					continue
				}

				r.l.Sugar().
					Infof("stopping of %d/%d instance of %s succeed!", idx+1, len(state.instances), taskName)
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
	r.l.Sugar().
		Infof("trying to update task instances of task %s to %d", taskName, instances)

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
			r.taskIDToState[taskName] = &taskState{}
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
		r.l.Sugar().
			Infof("updating of task instances of task %s from %d to %d is actually noop, skip", taskName, instances, instances)
		return nil
	}

	if len(state.instances) < instances {
		toAdd := instances - len(state.instances)

		r.l.Sugar().
			Infof("adding running additional %d tasks of %s", toAdd, taskName)
		for i := 0; i < toAdd; i++ {
			ctx, cancel := context.WithCancel(r.baseContext)
			instance := &taskInstance{
				ctx:     ctx,
				cancel:  cancel,
				errChan: make(chan error, 1),
			}

			go func() {
				for {
					err := task.Run(ctx)
					if err == nil {
						r.l.Sugar().
							Warnf("task %s unexpectedly ended it's job without an error, restarting", taskName)
						continue
					}

					// err is not nil
					if errors.Is(err, context.Canceled) ||
						errors.Is(err, context.DeadlineExceeded) {
						instance.errChan <- nil
						return
					} else {
						r.l.Sugar().Warnf("task %s unexpectedly failed: %s, restarting", taskName, err)
						continue
					}
				}
			}()

			state.instances = append(state.instances, instance)
		}

		r.l.Sugar().Infof("started new tasks, %d of %s running now", len(state.instances), taskName)

		return nil
	}

	// need to stop some instances of provided taskName
	toStop := len(state.instances) - instances
	r.l.Sugar().Infof("stopping %d tasks of %s", toStop, taskName)

	var errlist *multierror.Error
	for i := 0; i < toStop; i++ {
		taskInstance := state.instances[len(state.instances)-1]
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
