package checkerctrl

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/HazyCorp/govnilo/pkg/hazycheck"
	"github.com/HazyCorp/govnilo/pkg/hazyerr"
	"github.com/HazyCorp/govnilo/pkg/statestore"
)

type serviceRuntimeState struct {
	SLA             CheckerSLA
	CheckerDataPool [][]byte
}

func (s *serviceRuntimeState) Clone() serviceRuntimeState {
	dataPool := make([][]byte, len(s.CheckerDataPool))
	for idx, data := range s.CheckerDataPool {
		dataPool[idx] = make([]byte, len(data))
		copy(dataPool[idx], data)
	}

	return serviceRuntimeState{
		SLA:             s.SLA,
		CheckerDataPool: dataPool,
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type storageState struct {
	CheckerRuntimeStates map[hazycheck.CheckerID]serviceRuntimeState
	ControllerState      State

	initialized bool
}

// method is written to avoid nil reference exceptions
// on map usages
func (s *storageState) init() {
	if s.initialized {
		return
	}

	if s.CheckerRuntimeStates == nil {
		s.CheckerRuntimeStates = make(map[hazycheck.CheckerID]serviceRuntimeState)
	}
	if s.ControllerState.Services == nil {
		s.ControllerState.Services = make(map[string]ServiceState)
	}

	s.initialized = true
}

func (s *storageState) Clone() storageState {
	services := make(map[hazycheck.CheckerID]serviceRuntimeState)
	for k, v := range s.CheckerRuntimeStates {
		services[k] = v
	}

	return storageState{CheckerRuntimeStates: services, ControllerState: s.ControllerState.Clone()}
}

func (s *storageState) GetCheckerSLA(id hazycheck.CheckerID) (*CheckerSLA, error) {
	s.init()

	serviceState, exists := s.CheckerRuntimeStates[id]
	if !exists {
		return nil, errors.Errorf("cannot find service in file")
	}

	return &serviceState.SLA, nil
}

func (s *storageState) AppendCheck(
	id hazycheck.CheckerID,
	successfull bool,
) (*CheckerSLA, error) {
	s.init()

	serviceState, exists := s.CheckerRuntimeStates[id]
	if !exists {
		s.CheckerRuntimeStates[id] = serviceRuntimeState{}
	}

	sla := &serviceState.SLA
	if successfull {
		sla.SuccessfullAttempts++
	}
	sla.TotalAttempts++

	s.CheckerRuntimeStates[id] = serviceState
	return sla, nil
}

func (s *storageState) GetContestState() *State {
	s.init()

	clone := s.ControllerState.Clone()
	return &clone
}

func (s *storageState) SetContestState(newState *State) {
	s.init()

	s.ControllerState = newState.Clone()
}

func (s *storageState) AppendCheckerData(id hazycheck.CheckerID, data []byte) error {
	s.init()

	svc, exists := s.CheckerRuntimeStates[id]
	if !exists {
		s.CheckerRuntimeStates[id] = serviceRuntimeState{}
		svc = s.CheckerRuntimeStates[id]
	}

	svc.CheckerDataPool = append(svc.CheckerDataPool, data)
	s.CheckerRuntimeStates[id] = svc

	return nil
}

func (s *storageState) GetCheckerDataPool(id hazycheck.CheckerID) ([][]byte, error) {
	s.init()

	svc, exists := s.CheckerRuntimeStates[id]
	if !exists {
		s.CheckerRuntimeStates[id] = serviceRuntimeState{}
		svc = s.CheckerRuntimeStates[id]
	}
	svc = svc.Clone()

	return svc.CheckerDataPool, nil
}

func (s *storageState) RemoveDataFromPool(id hazycheck.CheckerID, idx uint64) ([]byte, error) {
	s.init()

	svc, exists := s.CheckerRuntimeStates[id]
	if !exists {
		return nil, errors.Wrap(hazyerr.ErrNotFound, "service not known")
	}
	svc = svc.Clone()

	if len(svc.CheckerDataPool) <= int(idx) {
		return nil, errors.Errorf("cannot remove data from pool, index out of range")
	}

	data := svc.CheckerDataPool[idx]
	svc.CheckerDataPool = append(svc.CheckerDataPool[:idx], svc.CheckerDataPool[idx+1:]...)

	return data, nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// FileStorage implements ControllerStorage
var _ ControllerStorage = &FileStorage{}

type FileStorage struct {
	mu sync.Mutex

	storage *statestore.JsonFile[storageState]
}

func NewFileStorage(path string) (*FileStorage, error) {
	internal := statestore.NewJsonFile[storageState](path)

	_, err := internal.RetrieveState(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "cannot setup json file storage")
	}

	return &FileStorage{storage: internal}, nil
}

func (s *FileStorage) GetCheckerSLA(ctx context.Context, checkerID hazycheck.CheckerID) (*CheckerSLA, error) {
	panic("implement me")
}

func (s *FileStorage) AppendCheck(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
	successfull bool,
) (*CheckerSLA, error) {
	var sla *CheckerSLA

	err := s.storage.UpdateState(ctx, func(st *storageState) error {
		currentSLA, err := st.AppendCheck(checkerID, successfull)
		if err != nil {
			return err
		}

		sla = currentSLA
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot append check to state")
	}

	return sla, nil
}

func (s *FileStorage) AppendCheckerData(ctx context.Context, checkerID hazycheck.CheckerID, data []byte) error {
	err := s.storage.UpdateState(ctx, func(st *storageState) error {
		return st.AppendCheckerData(checkerID, data)
	})
	if err != nil {
		return errors.Wrap(err, "cannot append checker data to state")
	}

	return nil
}

func (s *FileStorage) GetCheckerDataPool(ctx context.Context, checkerID hazycheck.CheckerID) ([][]byte, error) {
	// TODO: add caching
	state, err := s.storage.RetrieveState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve state from file")
	}

	return state.GetCheckerDataPool(checkerID)
}

func (s *FileStorage) RemoveDataFromPool(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
	idx uint64,
) ([]byte, error) {
	var data []byte

	err := s.storage.UpdateState(ctx, func(st *storageState) error {
		removedData, err := st.RemoveDataFromPool(checkerID, idx)
		if err != nil {
			return err
		}

		data = removedData
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot remove checker data from state")
	}

	return data, nil
}

func (s *FileStorage) GetContestState(ctx context.Context) (*State, error) {
	st, err := s.storage.RetrieveState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve state from file")
	}
	contestState := st.GetContestState()

	return contestState, nil
}

func (s *FileStorage) SetContestState(ctx context.Context, newState *State) error {
	err := s.storage.UpdateState(ctx, func(st *storageState) error {
		st.SetContestState(newState)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "cannot set contest state")
	}

	return nil
}

func (s *FileStorage) Flush(ctx context.Context) error {
	return nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type AsyncFileStoreConfig struct {
	Path         string        `yaml:"path" json:"path"`
	SyncInterval time.Duration `yaml:"sync_interval" json:"sync_interval"`
}

type AsyncFileStoreIn struct {
	fx.In

	Logger *zap.Logger
	Config AsyncFileStoreConfig
}

// AsyncFileStore implements Storage
var _ ControllerStorage = &AsyncFileStore{}

type AsyncFileStore struct {
	mu           sync.Mutex
	l            *zap.Logger
	storage      *statestore.JsonFile[storageState]
	currentState storageState
	conf         AsyncFileStoreConfig
}

func NewAsyncFileStore(in AsyncFileStoreIn) (*AsyncFileStore, error) {
	storage := statestore.NewJsonFile[storageState](in.Config.Path)
	state, err := storage.RetrieveState(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve initial state from file")
	}

	return &AsyncFileStore{
		l:            in.Logger,
		storage:      storage,
		currentState: state,
	}, nil
}

func NewAsyncFileStoreFX(in AsyncFileStoreIn, lc fx.Lifecycle) (*AsyncFileStore, error) {
	store, err := NewAsyncFileStore(in)
	if err != nil {
		return nil, err
	}

	runErrCh := make(chan error, 1)
	runCtx, runCancel := context.WithCancel(context.Background())

	lc.Append(fx.StartStopHook(
		func() {
			go func() {
				err := store.Run(runCtx)
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					err = nil
				}

				runErrCh <- err
			}()
		},
		func(ctx context.Context) error {
			runCancel()

			store.Flush(ctx)

			select {
			case <-ctx.Done():
				return errors.Wrap(ctx.Err(), "cannot await stopping async file storage")
			case err := <-runErrCh:
				return err
			}
		},
	))

	return store, nil
}

func (s *AsyncFileStore) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
			// pass
		}

		if err := s.Flush(ctx); err != nil {
			s.l.Error("cannot flush state to file", zap.Error(err))
		}
	}
}

func (s *AsyncFileStore) GetCheckerSLA(ctx context.Context, checkerID hazycheck.CheckerID) (*CheckerSLA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentState.GetCheckerSLA(checkerID)
}

func (s *AsyncFileStore) AppendCheck(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
	successfull bool,
) (*CheckerSLA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentState.AppendCheck(checkerID, successfull)
}

func (s *AsyncFileStore) AppendCheckerData(ctx context.Context, checkerID hazycheck.CheckerID, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentState.AppendCheckerData(checkerID, data)
}

func (s *AsyncFileStore) GetCheckerDataPool(ctx context.Context, checkerID hazycheck.CheckerID) ([][]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentState.GetCheckerDataPool(checkerID)
}

func (s *AsyncFileStore) RemoveDataFromPool(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
	idx uint64,
) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentState.RemoveDataFromPool(checkerID, idx)
}

func (s *AsyncFileStore) GetContestState(ctx context.Context) (*State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	st := s.currentState.GetContestState()
	return st, nil
}

func (s *AsyncFileStore) SetContestState(ctx context.Context, newState *State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentState.SetContestState(newState)
	return nil
}

func (s *AsyncFileStore) Flush(ctx context.Context) error {
	return s.storage.UpdateState(ctx, func(state *storageState) error {
		*state = s.currentState.Clone()
		return nil
	})
}
