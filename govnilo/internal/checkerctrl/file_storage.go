package checkerctrl

import (
	"context"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/HazyCorp/govnilo/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/govnilo/internal/statestore"
)

type internalState struct {
	ServiceToState map[string]*serviceState
	Settings       *Settings
}

func defaultInternalState() internalState {
	return internalState{
		ServiceToState: make(map[string]*serviceState),
		Settings:       &Settings{Services: make(map[string]*ServiceSettings)},
	}
}

func (s *internalState) Init() {
	s.Settings = &Settings{}
	s.Settings.Init()

	s.ServiceToState = make(map[string]*serviceState)
}

func (s *internalState) Clone() *internalState {
	if s == nil {
		return nil
	}

	st := &internalState{}
	st.Init()

	st.Settings = s.Settings.Clone()

	for svcName, state := range s.ServiceToState {
		st.ServiceToState[svcName] = state.Clone()
	}

	return st
}

type serviceState struct {
	CheckerToState map[string]*checkerState
}

func (s *serviceState) Init() {
	s.CheckerToState = make(map[string]*checkerState)
}

func (s *serviceState) Clone() *serviceState {
	if s == nil {
		return nil
	}

	st := &serviceState{}
	st.Init()

	for checkerName, checkerState := range s.CheckerToState {
		st.CheckerToState[checkerName] = checkerState.Clone()
	}

	return st
}

type checkerState struct {
	SLA             CheckerSLA
	checkerDataPool [][]byte
}

func (s *checkerState) Clone() *checkerState {
	if s == nil {
		return nil
	}

	st := &checkerState{SLA: s.SLA}
	st.checkerDataPool = make([][]byte, 0, len(s.checkerDataPool))

	for _, slice := range s.checkerDataPool {
		copied := make([]byte, len(slice))
		copy(copied, slice)
		st.checkerDataPool = append(st.checkerDataPool, copied)
	}

	return st
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type AsyncFileStoreConfig struct {
	Path         string        `yaml:"path"          json:"path"`
	SyncInterval time.Duration `yaml:"sync_interval" json:"sync_interval"`
}

type AsyncFileStoreIn struct {
	fx.In

	Logger *zap.Logger
	Config AsyncFileStoreConfig
}

type AsyncFileStore struct {
	mu        sync.RWMutex
	lastState *internalState
	store     *statestore.JsonFile[internalState]
	conf      AsyncFileStoreConfig
	l         *zap.Logger

	stopCh chan struct{}
}

func NewAsyncFileStore(in AsyncFileStoreIn) (*AsyncFileStore, error) {
	store := statestore.NewJsonFile(in.Config.Path, defaultInternalState)

	lastState, err := store.RetrieveState(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve the last state after restart")
	}

	return &AsyncFileStore{
		lastState: &lastState,
		store:     store,
		l:         in.Logger,
		conf:      in.Config,

		stopCh: make(chan struct{}),
	}, nil
}

func NewAsyncFileStoreFX(in AsyncFileStoreIn, lc fx.Lifecycle) (*AsyncFileStore, error) {
	store, err := NewAsyncFileStore(in)
	if err != nil {
		return nil, err
	}

	lc.Append(fx.StartStopHook(
		store.Start,
		store.Stop,
	))

	return store, nil
}

func (s *AsyncFileStore) Stop(ctx context.Context) error {
	close(s.stopCh)
	return s.Flush(ctx)
}

func (s *AsyncFileStore) Start() {
	go s.run()
}

func (s *AsyncFileStore) run() {
	for {
		select {
		case <-time.After(s.conf.SyncInterval):
			// pass
		case <-s.stopCh:
			return
		}

		s.l.Info("flushing the checker state to the disk")
		if err := s.Flush(context.TODO()); err != nil {
			s.l.Error("cannot flush the checker state to the disk", zap.Error(err))
		}
	}
}

func (s *AsyncFileStore) GetCheckerSLA(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
) (*CheckerSLA, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	checkerState, err := s.getCheckerStateLocked(checkerID)
	if err != nil {
		return nil, errors.Wrap(err, "cannot find the checker state")
	}

	// copy the sla
	sla := checkerState.SLA
	return &sla, nil
}

// TODO: add weights
func (s *AsyncFileStore) AppendCheck(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
	successfull bool,
) (*CheckerSLA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	checkerState, err := s.getOrCreateCheckerStateLocked(checkerID)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get the checker state")
	}

	// TODO: add weights
	checkerState.SLA.TotalAttempts += 1
	if successfull {
		checkerState.SLA.SuccessfullAttempts += 1
	}

	// copy the sla
	sla := checkerState.SLA
	return &sla, nil
}

func (s *AsyncFileStore) AppendCheckerData(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
	data []byte,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	checkerState, err := s.getOrCreateCheckerStateLocked(checkerID)
	if err != nil {
		return errors.Wrap(err, "cannot get the checker state")
	}

	checkerState.checkerDataPool = append(checkerState.checkerDataPool, data)
	return nil
}

// TODO: use more synchronious interface (need to delete entries under the lock)
func (s *AsyncFileStore) GetCheckerDataPool(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
) ([][]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	checkerState, err := s.getCheckerStateLocked(checkerID)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get the checker state")
	}

	// need to copy these slices to avoid changing them from outside the function
	ret := make([][]byte, 0, len(checkerState.checkerDataPool))
	for _, slice := range checkerState.checkerDataPool {
		sliceCopy := make([]byte, len(slice))
		copy(sliceCopy, slice)

		ret = append(ret, sliceCopy)
	}

	return ret, nil
}

func (s *AsyncFileStore) RemoveDataFromPool(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
	idx uint64,
) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	checkerState, err := s.getOrCreateCheckerStateLocked(checkerID)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get the checker state")
	}

	dataPool := checkerState.checkerDataPool
	if idx >= uint64(len(dataPool)) {
		return nil, errors.New("index out of bounds")
	}

	ret := dataPool[idx]
	checkerState.checkerDataPool = append(dataPool[:idx], dataPool[idx+1:]...)

	return ret, nil
}

func (s *AsyncFileStore) GetContestState(ctx context.Context) (*Settings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.lastState.Settings.Clone(), nil
}

func (s *AsyncFileStore) SetContestState(ctx context.Context, newSettings *Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastState.Settings = newSettings.Clone()
	return nil
}

func (s *AsyncFileStore) Flush(ctx context.Context) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.store.UpdateState(ctx, func(currentState *internalState) error {
		copied := s.lastState.Clone()
		*currentState = *copied

		return nil
	})
}

func (s *AsyncFileStore) getCheckerStateLocked(
	checkerID hazycheck.CheckerID,
) (*checkerState, error) {
	svcState, found := s.lastState.ServiceToState[checkerID.Service]
	if !found {
		return nil, errors.Errorf("cannot find state of the %+v checker", checkerID)
	}

	chckrState, found := svcState.CheckerToState[checkerID.Name]
	if !found {
		return nil, errors.Errorf("cannot find state of the %+v checker", checkerID)
	}

	return chckrState, nil
}

func (s *AsyncFileStore) getOrCreateCheckerStateLocked(
	checkerID hazycheck.CheckerID,
) (*checkerState, error) {
	svcState, found := s.lastState.ServiceToState[checkerID.Service]
	if !found {
		newState := &serviceState{}
		newState.Init()

		s.lastState.ServiceToState[checkerID.Service] = newState
		svcState = newState
	}

	chckrState, found := svcState.CheckerToState[checkerID.Name]
	if !found {
		newState := &checkerState{}
		svcState.CheckerToState[checkerID.Name] = newState

		chckrState = newState
	}

	return chckrState, nil
}
