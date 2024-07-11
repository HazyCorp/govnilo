package checkerctrl

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/HazyCorp/checker/pkg/hazyerr"
	"github.com/HazyCorp/checker/pkg/statestore"
)

type serviceRuntimeState struct {
	SLA             ServiceSLA
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
	Services map[string]serviceRuntimeState
	Routines ServicesContestState

	initOnce sync.Once
}

func (s *storageState) init() {
	s.initOnce.Do(func() {
		if s.Services == nil {
			s.Services = make(map[string]serviceRuntimeState)
		}
		if s.Routines.Services == nil {
			s.Routines.Services = make(map[string]ServiceCheckerState)
		}
	})
}

func (s *storageState) Clone() storageState {
	services := make(map[string]serviceRuntimeState)
	for k, v := range s.Services {
		services[k] = v
	}

	return storageState{Services: services, Routines: s.Routines.Clone()}
}

func (s *storageState) GetServiceSLA(serviceName string) (*ServiceSLA, error) {
	s.init()

	serviceState, exists := s.Services[serviceName]
	if !exists {
		return nil, errors.Errorf("cannot find service in file")
	}

	return &serviceState.SLA, nil
}

func (s *storageState) AppendServiceCheck(
	serviceName string,
	successfull bool,
) (*ServiceSLA, error) {
	s.init()

	serviceState, exists := s.Services[serviceName]
	if !exists {
		s.Services[serviceName] = serviceRuntimeState{}
	}

	sla := &serviceState.SLA
	if successfull {
		sla.SuccessfullAttempts++
	}
	sla.TotalAttempts++

	s.Services[serviceName] = serviceState
	return sla, nil
}

func (s *storageState) GetContestState() *ServicesContestState {
	s.init()

	clone := s.Routines.Clone()
	return &clone
}

func (s *storageState) SetContestState(newState *ServicesContestState) {
	s.init()

	s.Routines = newState.Clone()
}

func (s *storageState) AppendCheckerData(serviceName string, data []byte) error {
	s.init()

	svc, exists := s.Services[serviceName]
	if !exists {
		s.Services[serviceName] = serviceRuntimeState{}
		svc = s.Services[serviceName]
	}

	svc.CheckerDataPool = append(svc.CheckerDataPool, data)
	s.Services[serviceName] = svc

	return nil
}

func (s *storageState) GetCheckerDataPool(serviceName string) ([][]byte, error) {
	s.init()

	svc, exists := s.Services[serviceName]
	if !exists {
		s.Services[serviceName] = serviceRuntimeState{}
		svc = s.Services[serviceName]
	}
	svc = svc.Clone()

	return svc.CheckerDataPool, nil
}

func (s *storageState) RemoveDataFromPool(serviceName string, idx uint64) ([]byte, error) {
	s.init()

	svc, exists := s.Services[serviceName]
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

func (s *FileStorage) GetServiceSLA(ctx context.Context, serviceName string) (*ServiceSLA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.storage.RetrieveState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve state from file")
	}

	return state.GetServiceSLA(serviceName)
}

func (s *FileStorage) AppendServiceCheck(
	ctx context.Context,
	serviceName string,
	successfull bool,
) (*ServiceSLA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var returnSLA ServiceSLA
	err := s.storage.UpdateState(ctx, func(state *storageState) error {
		sla, err := state.AppendServiceCheck(serviceName, successfull)
		if err != nil {
			return err
		}

		returnSLA = *sla
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot update state in file storage")
	}

	return &returnSLA, nil
}

func (s *FileStorage) AppendCheckerData(
	ctx context.Context,
	serviceName string,
	data []byte,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.storage.UpdateState(ctx, func(state *storageState) error {
		return state.AppendCheckerData(serviceName, data)
	})
}

func (s *FileStorage) GetCheckerDataPool(
	ctx context.Context,
	serviceName string,
) ([][]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.storage.RetrieveState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve state from file storage")
	}

	return state.GetCheckerDataPool(serviceName)
}

func (s *FileStorage) RemoveDataFromPool(
	ctx context.Context,
	servicieName string,
	idx uint64,
) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var data []byte
	err := s.storage.UpdateState(ctx, func(state *storageState) error {
		d, err := state.RemoveDataFromPool(servicieName, idx)
		if err != nil {
			return errors.Wrap(err, "cannot remove data from pool")
		}

		data = d
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "cannot update state in file storage")
	}

	return data, nil
}

func (s *FileStorage) GetContestState(ctx context.Context) (*ServicesContestState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.storage.RetrieveState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve state from file")
	}

	return state.GetContestState(), nil
}

func (s *FileStorage) SetContestState(ctx context.Context, newState *ServicesContestState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.storage.UpdateState(ctx, func(state *storageState) error {
		state.SetContestState(newState)
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "cannot update state in file store")
	}

	return nil
}

func (s *FileStorage) Flush(ctx context.Context) error {
	return nil
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type AsyncFileStoreConfig struct {
	Path string `yaml:"path" json:"path"`
}

type AsyncFileStoreIn struct {
	fx.In

	Logger *zap.Logger
	Config AsyncFileStoreConfig
}

var _ ControllerStorage = &AsyncFileStore{}

type AsyncFileStore struct {
	mu           sync.Mutex
	l            *zap.Logger
	storage      *statestore.JsonFile[storageState]
	currentState storageState
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

func (s *AsyncFileStore) GetServiceSLA(
	ctx context.Context,
	serviceName string,
) (*ServiceSLA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentState.GetServiceSLA(serviceName)
}

func (s *AsyncFileStore) AppendServiceCheck(
	ctx context.Context,
	serviceName string,
	successfull bool,
) (*ServiceSLA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentState.AppendServiceCheck(serviceName, successfull)
}

func (s *AsyncFileStore) GetContestState(ctx context.Context) (*ServicesContestState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentState.GetContestState(), nil
}

func (s *AsyncFileStore) SetContestState(
	ctx context.Context,
	newState *ServicesContestState,
) error {
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

func (s *AsyncFileStore) AppendCheckerData(
	ctx context.Context,
	serviceName string,
	data []byte,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentState.AppendCheckerData(serviceName, data)
}

func (s *AsyncFileStore) GetCheckerDataPool(
	ctx context.Context,
	serviceName string,
) ([][]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentState.GetCheckerDataPool(serviceName)
}

func (s *AsyncFileStore) RemoveDataFromPool(
	ctx context.Context,
	servicieName string,
	idx uint64,
) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentState.RemoveDataFromPool(servicieName, idx)
}
