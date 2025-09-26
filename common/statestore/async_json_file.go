package statestore

import (
	"context"
	"github.com/HazyCorp/govnilo/common/hzlog"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type AsyncJsonFileConfig struct {
	Path         string        `json:"path" yaml:"path"`
	SyncInterval time.Duration `json:"sync_interval" yaml:"sync_interval"`
}

type AsyncJsonFile[TState any] struct {
	mu        sync.Mutex
	l         *slog.Logger
	c         AsyncJsonFileConfig
	lastState TState
	syncState *SyncJsonFile[TState]
}

func NewAsyncJsonFile[TState any](c AsyncJsonFileConfig, defaultFunc defaultFunc[TState], opts ...StateStoreOpt) (StateStore[TState], error) {
	var o stateStoreOptions
	for _, opt := range opts {
		opt.apply(&o)
	}

	var logger *slog.Logger
	if o.logger != nil {
		logger = o.logger
	} else {
		logger = hzlog.NopLogger()
	}
	logger = logger.With(
		slog.String("component", "async_json_file_store"),
		slog.String("path", c.Path),
	)

	syncJsonFile := NewSyncJsonFile(SyncJsonFileConfig{Path: c.Path}, defaultFunc)
	curState, err := syncJsonFile.RetrieveState(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve current state from sync file storage")
	}

	store := AsyncJsonFile[TState]{
		c:         c,
		l:         logger,
		lastState: curState,
		syncState: syncJsonFile,
	}
	go store.syncRoutine()

	return &store, nil
}

func (j *AsyncJsonFile[TState]) RetrieveState(ctx context.Context) (TState, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	return j.lastState, nil
}

func (j *AsyncJsonFile[TState]) UpdateState(ctx context.Context, updateF func(s *TState) error) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	if err := updateF(&j.lastState); err != nil {
		return errors.Wrap(err, "cannot update state, updateF errored")
	}

	return nil
}

func (j *AsyncJsonFile[TState]) Flush(ctx context.Context) error {
	return j.sync(ctx)
}

func (j *AsyncJsonFile[TState]) sync(ctx context.Context) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	err := j.syncState.UpdateState(ctx, func(s *TState) error {
		*s = j.lastState
		return nil
	})

	return err
}

func (j *AsyncJsonFile[TState]) syncRoutine() {
	ticker := time.NewTicker(j.c.SyncInterval)

	for range ticker.C {
		if err := j.sync(context.Background()); err != nil {
			j.l.Warn("cannot sync the state to the file", hzlog.Error(err))
		}
	}
}
