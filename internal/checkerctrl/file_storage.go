package checkerctrl

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/pkg/common/checkersettings"
	"github.com/HazyCorp/govnilo/pkg/common/statestore"

	"github.com/pkg/errors"
	"go.uber.org/fx"
)

type internalState struct {
	ServiceToState map[string]*serviceState
	Settings       *checkersettings.Settings
}

func defaultInternalState() internalState {
	return internalState{
		ServiceToState: make(map[string]*serviceState),
		Settings:       &checkersettings.Settings{Services: make(map[string]*checkersettings.ServiceSettings)},
	}
}

func (s *internalState) Init() {
	s.Settings = &checkersettings.Settings{}
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
	SLA             CheckerPointsStats
	CheckerDataPool []DataRecord
}

func (s *checkerState) Clone() *checkerState {
	if s == nil {
		return nil
	}

	st := &checkerState{SLA: s.SLA}
	st.CheckerDataPool = make([]DataRecord, 0, len(s.CheckerDataPool))

	for _, rec := range s.CheckerDataPool {
		copiedData := make([]byte, len(rec.Data))
		copy(copiedData, rec.Data)
		copiedRecord := DataRecord{Data: copiedData, Created: rec.Created}
		st.CheckerDataPool = append(st.CheckerDataPool, copiedRecord)
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

	Logger *slog.Logger
	Config AsyncFileStoreConfig
}

type AsyncFileStore struct {
	store statestore.StateStore[internalState]
	conf  AsyncFileStoreConfig
	l     *slog.Logger

	mu sync.RWMutex
}

func NewAsyncFileStore(in AsyncFileStoreIn) (*AsyncFileStore, error) {
	// TODO: use slog.Logger
	store, err := statestore.NewAsyncJsonFile(
		statestore.AsyncJsonFileConfig{
			Path:         in.Config.Path,
			SyncInterval: in.Config.SyncInterval,
		},
		defaultInternalState,
		statestore.WithLogger(slog.Default()),
	)

	if err != nil {
		return nil, errors.Wrap(err, "cannot create async file storage")
	}

	return &AsyncFileStore{
		store: store,
		l:     in.Logger,
		conf:  in.Config,
	}, nil
}

func NewAsyncFileStoreFX(in AsyncFileStoreIn) (*AsyncFileStore, error) {
	store, err := NewAsyncFileStore(in)
	if err != nil {
		return nil, err
	}

	return store, nil
}

func (s *AsyncFileStore) AppendCheckerData(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
	data []byte,
) error {
	err := s.store.UpdateState(ctx, func(state *internalState) error {
		checkerState := s.getOrCreateCheckerState(state, checkerID)
		record := DataRecord{Data: data, Created: time.Now()}
		checkerState.CheckerDataPool = append(checkerState.CheckerDataPool, record)

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "cannot update the state in internal store")
	}

	return nil
}

func (s *AsyncFileStore) GetCheckerDataPool(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
) ([]DataRecord, error) {
	state, err := s.store.RetrieveState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve state from internal store")
	}

	checkerState := s.getOrCreateCheckerState(&state, checkerID)

	// don't copy the slice, user is not allowed to change slices
	return checkerState.CheckerDataPool, nil
}

func (s *AsyncFileStore) RemoveDataFromPool(
	ctx context.Context,
	checkerID hazycheck.CheckerID,
	f NeedDeleteFunc,
) error {
	err := s.store.UpdateState(ctx, func(state *internalState) error {
		checkerState := s.getOrCreateCheckerState(state, checkerID)

		dataPool := checkerState.CheckerDataPool
		dataPool = slices.DeleteFunc(dataPool, func(rec DataRecord) bool { return f(&rec) })

		checkerState.CheckerDataPool = dataPool
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "cannot update the state in internal storage")
	}

	return nil
}

func (s *AsyncFileStore) Flush(ctx context.Context) error {
	if err := s.store.Flush(ctx); err != nil {
		return errors.Wrap(err, "cannot flush internal state")
	}

	return nil
}

func (s *AsyncFileStore) getOrCreateCheckerState(
	state *internalState,
	checkerID hazycheck.CheckerID,
) *checkerState {
	svcState, found := state.ServiceToState[checkerID.Service]
	if !found {
		newState := &serviceState{}
		newState.Init()

		state.ServiceToState[checkerID.Service] = newState
		svcState = newState
	}

	chckrState, found := svcState.CheckerToState[checkerID.Name]
	if !found {
		newState := &checkerState{}
		svcState.CheckerToState[checkerID.Name] = newState

		chckrState = newState
	}

	return chckrState
}
