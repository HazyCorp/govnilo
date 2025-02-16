package statestore

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/pkg/errors"
)

type defaultFunc[TState any] func() TState

type JsonFile[TState any] struct {
	mu sync.Mutex

	defaultVal defaultFunc[TState]
	path       string
}

func NewJsonFile[TState any](path string, defaultVal defaultFunc[TState]) *JsonFile[TState] {
	return &JsonFile[TState]{path: path, defaultVal: defaultVal}
}

func (j *JsonFile[TState]) RetrieveState(ctx context.Context) (TState, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	// TODO: add inmemory caching

	return j.retrieveState()
}

// operationWithState edits the state in file and atomically update it in file
func (j *JsonFile[TState]) UpdateState(ctx context.Context, updateF func(s *TState) error) error {
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

	if err := os.Rename(tmpPath, j.path); err != nil {
		return errors.Wrap(err, "cannot move temporary file in place of the main state file")
	}

	return nil
}

// writeStateToTemp returns name of the tempfile
func (j *JsonFile[TState]) writeStateToTemp(st *TState) (string, error) {
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

func (j *JsonFile[TState]) operationWithFile(f func(f *os.File) error) error {
	file, err := os.OpenFile(j.path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return errors.Wrapf(err, "cannot open file %q", j.path)
	}
	defer file.Close()

	if err := f(file); err != nil {
		return err
	}

	return nil
}

func (j *JsonFile[TState]) retrieveState() (TState, error) {
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
