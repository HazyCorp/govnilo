package checkerctrl

import (
	"context"
	"maps"
	"time"

	"github.com/HazyCorp/checker/pkg/hazycheck"
)

// example state (represented in yaml)
// ...
// state:
//   services:
//     example:
//       target: 'localhost:8080'
//       checkers:
//         simple_user_flow:
//           check:
//             rate:
//               times: 500
//               per: 1s
//           get:
//             rate:
//               times: 5
//               per: 1s
//         user_with_big_payload:
//           check:
//             rate:
//               times: 5
//               per: 1m
//            get:
//              rate:
//                times: 1
//                per: 1h
//       sploits:
//         drop_database_sql_injection:
//           rate:
//             times: 1
//             per: 1s

type Rate struct {
	Times uint64
	Per   time.Duration
}

type SploitState struct {
	Rate Rate
}

type CheckerState struct {
	Check CheckerCheckState
	Get   CheckerGetState
}

type CheckerGetState struct {
	Rate Rate
}

type CheckerCheckState struct {
	Rate Rate
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ServiceState struct {
	Target   string
	Checkers map[string]CheckerState
	Sploits  map[string]SploitState
}

func (s *ServiceState) Clone() ServiceState {
	return ServiceState{
		Target:   s.Target,
		Checkers: maps.Clone(s.Checkers),
		Sploits:  maps.Clone(s.Sploits),
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type State struct {
	Services map[string]ServiceState
}

func (s *State) Clone() State {
	services := make(map[string]ServiceState, len(s.Services))
	for k, v := range s.Services {
		services[k] = v.Clone()
	}

	return State{Services: services}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ServiceSLA struct {
	TotalAttempts       int
	SuccessfullAttempts int
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ControllerStorage interface {
	GetCheckerSLA(ctx context.Context, checkerID hazycheck.CheckerID) (*ServiceSLA, error)
	AppendCheck(
		ctx context.Context,
		checkerID hazycheck.CheckerID,
		successfull bool,
	) (*ServiceSLA, error)

	AppendCheckerData(ctx context.Context, checkerID hazycheck.CheckerID, data []byte) error
	GetCheckerDataPool(ctx context.Context, checkerID hazycheck.CheckerID) ([][]byte, error)
	RemoveDataFromPool(
		ctx context.Context,
		checkerID hazycheck.CheckerID,
		idx uint64,
	) ([]byte, error)

	GetContestState(ctx context.Context) (*State, error)
	SetContestState(ctx context.Context, newState *State) error

	Flush(ctx context.Context) error
}
