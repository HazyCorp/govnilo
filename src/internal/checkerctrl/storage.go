package checkerctrl

import (
	"context"
	"time"
)

type Rate struct {
	Times uint64
	Per   time.Duration
}

type SploitState struct {
	Rate Rate
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ServiceCheckerState struct {
	Target    string
	CheckRate Rate
	GetRate   Rate
	Sploits   map[string]SploitState
}

func (s *ServiceCheckerState) Clone() ServiceCheckerState {
	sploits := make(map[string]SploitState, len(s.Sploits))
	for k, v := range s.Sploits {
		sploits[k] = v
	}

	return ServiceCheckerState{
		CheckRate: s.CheckRate,
		GetRate:   s.GetRate,
		Sploits:   sploits,
		Target:    s.Target,
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ServicesContestState struct {
	Services map[string]ServiceCheckerState
}

func (s *ServicesContestState) Clone() ServicesContestState {
	services := make(map[string]ServiceCheckerState, len(s.Services))
	for k, v := range s.Services {
		services[k] = v.Clone()
	}

	return ServicesContestState{Services: services}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ServiceSLA struct {
	TotalAttempts       int
	SuccessfullAttempts int
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ControllerStorage interface {
	GetServiceSLA(ctx context.Context, serviceName string) (*ServiceSLA, error)
	AppendServiceCheck(
		ctx context.Context,
		serviceName string,
		successfull bool,
	) (*ServiceSLA, error)

	AppendCheckerData(ctx context.Context, serviceName string, data []byte) error
	GetCheckerDataPool(ctx context.Context, serviceName string) ([][]byte, error)
	RemoveDataFromPool(ctx context.Context, servicieName string, idx uint64) ([]byte, error)

	GetContestState(ctx context.Context) (*ServicesContestState, error)
	SetContestState(ctx context.Context, newState *ServicesContestState) error

	Flush(ctx context.Context) error
}
