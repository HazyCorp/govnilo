package util

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
)

type StateStorage interface {
	UpdateState(state interface{}) error
	GetAndUpdate(updateFunc func(state []byte) (interface{}, error)) error
	GetState() ([]byte, error)
}

type FileStateStorage struct {
	mut  sync.Mutex
	path string
}

func NewFileStateStorage(statePath string) (StateStorage, error) {
	err := os.MkdirAll(path.Dir(statePath), 0666)
	if err != nil {
		return nil, fmt.Errorf("failed create state dir %w", err)
	}

	return &FileStateStorage{mut: sync.Mutex{}, path: statePath}, nil
}

func (f *FileStateStorage) UpdateState(state interface{}) error {
	f.mut.Lock()
	defer f.mut.Unlock()
	marshal, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed marshal state %w", err)
	}

	err = os.WriteFile(f.path, marshal, 0666)
	if err != nil {
		return fmt.Errorf("failed write state %w", err)
	}

	return nil
}

func (f *FileStateStorage) GetAndUpdate(updateFunc func(state []byte) (interface{}, error)) error {
	f.mut.Lock()
	defer f.mut.Unlock()

	data, err := os.ReadFile(f.path)
	if err != nil {
		return fmt.Errorf("failed read state %w", err)
	}

	state, err := updateFunc(data)
	if err != nil {
		return fmt.Errorf("failed updateFunc %w", err)
	}

	marshal, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed marshal state %w", err)
	}

	err = os.WriteFile(f.path, marshal, 0666)
	if err != nil {
		return fmt.Errorf("failed write state %w", err)
	}

	return nil
}

func (f *FileStateStorage) GetState() ([]byte, error) {
	f.mut.Lock()
	defer f.mut.Unlock()

	data, err := os.ReadFile(f.path)
	if err != nil {
		return nil, fmt.Errorf("failed read state %w", err)
	}

	return data, nil
}

func ParseJson[T interface{}](data []byte) (T, error) {
	result := new(T)

	err := json.Unmarshal(data, &result)
	if err != nil {
		return *result, err
	}

	return *result, nil
}
