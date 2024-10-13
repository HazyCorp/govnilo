package checkerctrl

import (
	"context"
	"maps"
	"time"

	"github.com/HazyCorp/govnilo/govnilo/internal/hazycheck"
)

// example settings (represented in yaml)
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

type SploitSettings struct {
	Rate Rate
}

type CheckerSettings struct {
	Check CheckerCheckSettings
	Get   CheckerGetSettings
}

type CheckerGetSettings struct {
	Rate Rate
}

type CheckerCheckSettings struct {
	Rate Rate
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ServiceSettings struct {
	Target   string
	Checkers map[string]*CheckerSettings
	Sploits  map[string]*SploitSettings
}

func (s *ServiceSettings) Init() {
	s.Checkers = make(map[string]*CheckerSettings)
	s.Sploits = make(map[string]*SploitSettings)
}

func (s *ServiceSettings) Clone() ServiceSettings {
	if s == nil {
		return ServiceSettings{}
	}

	return ServiceSettings{
		Target:   s.Target,
		Checkers: maps.Clone(s.Checkers),
		Sploits:  maps.Clone(s.Sploits),
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type Settings struct {
	Services map[string]*ServiceSettings
}

func (s *Settings) Init() {
	s.Services = make(map[string]*ServiceSettings)
}

func (s *Settings) Clone() *Settings {
	if s == nil {
		return nil
	}

	services := make(map[string]*ServiceSettings, len(s.Services))
	for k, v := range s.Services {
		svc := v.Clone()
		services[k] = &svc
	}

	return &Settings{Services: services}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type CheckerSLA struct {
	TotalAttempts       int
	SuccessfullAttempts int
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ControllerStorage interface {
	GetCheckerSLA(ctx context.Context, checkerID hazycheck.CheckerID) (*CheckerSLA, error)
	AppendCheck(
		ctx context.Context,
		checkerID hazycheck.CheckerID,
		successfull bool,
	) (*CheckerSLA, error)

	AppendCheckerData(ctx context.Context, checkerID hazycheck.CheckerID, data []byte) error
	GetCheckerDataPool(ctx context.Context, checkerID hazycheck.CheckerID) ([][]byte, error)
	RemoveDataFromPool(
		ctx context.Context,
		checkerID hazycheck.CheckerID,
		idx uint64,
	) ([]byte, error)

	GetContestState(ctx context.Context) (*Settings, error)
	SetContestState(ctx context.Context, newState *Settings) error

	Flush(ctx context.Context) error
}
