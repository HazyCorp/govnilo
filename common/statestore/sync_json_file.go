package statestore

import (
	"context"
	"encoding/json"
	"github.com/HazyCorp/govnilo/common/hzlog"
	"log/slog"
	"os"
	"sync"

	"github.com/pkg/errors"
)

type SyncJsonFileConfig struct {
	Path string `json:"path" yaml:"path"`
}

type SyncJsonFile[TState any] struct {
	mu sync.Mutex

	defaultVal defaultFunc[TState]
	c          SyncJsonFileConfig
	l          *slog.Logger
}

func NewSyncJsonFile[TState any](c SyncJsonFileConfig, defaultVal defaultFunc[TState], opts ...StateStoreOpt) *SyncJsonFile[TState] {
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
		slog.String("component", "sync_json_file_store"),
		slog.String("path", c.Path),
	)

	return &SyncJsonFile[TState]{c: c, defaultVal: defaultVal, l: logger}
}

func (j *SyncJsonFile[TState]) RetrieveState(ctx context.Context) (TState, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	// TODO: add inmemory caching
	// (we may return last state from memory, while no write operations appeared)

	return j.retrieveState()
}

// operationWithState edits the state in file and atomically update it in file
func (j *SyncJsonFile[TState]) UpdateState(ctx context.Context, updateF func(s *TState) error) error {
	j.mu.Lock()
	defer j.mu.Unlock()

	st, err := j.retrieveState()
	if err != nil {
		return errors.Wrap(err, "cannot retrieve state for following changes")
	}

	if err := updateF(&st); err != nil {
		return errors.Wrap(err, "cannot change the state")
	}

	tmpPath, err := j.writeStateToTemp(&st)
	if err != nil {
		return errors.Wrap(err, "cannot write updated state to temporary file")
	}

	if err := os.Rename(tmpPath, j.c.Path); err != nil {
		return errors.Wrap(err, "cannot move temporary file in place of the main state file")
	}

	return nil
}

func (j *SyncJsonFile[TState]) Flush(ctx context.Context) error {
	return nil
}

// writeStateToTemp returns name of the tempfile
func (j *SyncJsonFile[TState]) writeStateToTemp(st *TState) (string, error) {
	tmp, err := os.CreateTemp("/tmp/", "json_file_storage")
	if err != nil {
		return "", errors.Wrap(err, "cannot open temporary file to make changes atomically")
	}
	defer tmp.Close()

	if err := json.NewEncoder(tmp).Encode(&st); err != nil {
		return "", errors.Wrap(err, "unreachable: cannot encode new state in json format")
	}

	return tmp.Name(), nil
}

func (j *SyncJsonFile[TState]) operationWithFile(f func(f *os.File) error) error {
	file, err := os.OpenFile(j.c.Path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrapf(err, "cannot open file %q", j.c.Path)
	}
	defer file.Close()

	if err := f(file); err != nil {
		return err
	}

	return nil
}

func (j *SyncJsonFile[TState]) retrieveState() (TState, error) {
	state := j.defaultVal()

	err := j.operationWithFile(func(f *os.File) error {
		stat, err := f.Stat()
		if err != nil {
			return errors.Wrap(err, "cannot stat file")
		}

		if stat.Size() == 0 {
			// file is empty, dont change state
			return nil
		}

		return json.NewDecoder(f).Decode(&state)
	})
	if err != nil {
		return state, errors.Wrap(err, "cannot retrieve state from file")
	}

	return state, nil
}
