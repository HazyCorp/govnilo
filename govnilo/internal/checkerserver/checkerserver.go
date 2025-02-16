package checkerserver

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/HazyCorp/govnilo/govnilo/internal/checkerctrl"
	"github.com/HazyCorp/govnilo/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/govnilo/internal/pb"
)

type CheckerServer struct {
	pb.UnimplementedCheckerServiceServer

	storage  checkerctrl.ControllerStorage
	checkers map[hazycheck.CheckerID]hazycheck.Checker
	sploits  map[hazycheck.SploitID]hazycheck.Sploit
	l        *zap.Logger
}

type CheckerServerIn struct {
	fx.In

	Storage  checkerctrl.ControllerStorage
	Checkers []hazycheck.Checker `group:"checkers"`
	Sploits  []hazycheck.Sploit  `group:"sploits"`
	Logger   *zap.Logger
}

func New(in CheckerServerIn) *CheckerServer {
	checkers := make(map[hazycheck.CheckerID]hazycheck.Checker, len(in.Checkers))
	for _, c := range in.Checkers {
		checkers[c.CheckerID()] = c
	}

	sploits := make(map[hazycheck.SploitID]hazycheck.Sploit, len(in.Sploits))
	for _, s := range in.Sploits {
		sploits[s.SploitID()] = s
	}

	return &CheckerServer{
		storage:  in.Storage,
		checkers: checkers,
		sploits:  sploits,
		l:        in.Logger,
	}
}

func NewFX(in CheckerServerIn, srv *grpc.Server) *CheckerServer {
	s := New(in)
	pb.RegisterCheckerServiceServer(srv, s)
	reflection.Register(srv)

	return s
}

func (s *CheckerServer) SetState(
	ctx context.Context,
	in *pb.SetStateReq,
) (*pb.SetStateRsp, error) {
	converted := pbToState(in.GetState())
	for svcName, svcState := range converted.Services {
		for checkerName := range svcState.Checkers {
			checkerID := hazycheck.CheckerID{
				Service: svcName,
				Name:    checkerName,
			}

			if _, exists := s.checkers[checkerID]; !exists {
				return nil, errors.Errorf(
					"checker %+v not registered, config is invalid",
					checkerID,
				)
			}
		}

		for sploitName := range svcState.Sploits {
			sploitID := hazycheck.SploitID{
				Service: svcName,
				Name:    sploitName,
			}

			if _, exists := s.sploits[sploitID]; !exists {
				return nil, errors.Errorf("sploit %+v not registered, config is invalid", sploitID)
			}
		}
	}

	if err := s.storage.SetContestState(ctx, &converted); err != nil {
		return nil, errors.Wrap(err, "cannot save state in storage")
	}

	return &pb.SetStateRsp{}, nil
}

func (s *CheckerServer) GetState(
	ctx context.Context,
	in *pb.GetStateReq,
) (*pb.GetStateRsp, error) {
	state, err := s.storage.GetContestState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve state from storage")
	}

	return &pb.GetStateRsp{State: stateToPb(state)}, nil
}

func (s *CheckerServer) GetSLA(ctx context.Context, req *pb.GetSLAReq) (*pb.GetSLAResponse, error) {
	slas := make(map[string]*pb.ServiceSLA, len(s.checkers))
	for checkerID := range s.checkers {
		sla, err := s.storage.GetCheckerSLA(ctx, checkerID)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get sla for checker %+v from storage", checkerID)
		}

		svcSLA, found := slas[checkerID.Service]
		if !found {
			svcSLA = &pb.ServiceSLA{
				Slas: make(map[string]*pb.SLA),
			}
			slas[checkerID.Service] = svcSLA
		}

		svcSLA.Slas[checkerID.Name] = &pb.SLA{
			SuccessCount: uint64(sla.SuccessfullAttempts),
			TotalCount:   uint64(sla.TotalAttempts),
		}
	}

	return &pb.GetSLAResponse{
		Slas: slas,
	}, nil
}

func pbToState(in *pb.State) checkerctrl.Settings {
	services := make(map[string]*checkerctrl.ServiceSettings, len(in.Services))
	for name, state := range in.GetServices() {
		serviceState := pbToCheckerState(state)
		services[name] = serviceState
	}

	return checkerctrl.Settings{
		Services: services,
	}
}

func stateToPb(in *checkerctrl.Settings) *pb.State {
	services := make(map[string]*pb.ServiceState, len(in.Services))
	for name, state := range in.Services {
		pbState := checkerStateToPB(state)
		services[name] = pbState
	}

	return &pb.State{
		Services: services,
	}
}

func pbToRate(r *pb.Rate) checkerctrl.Rate {
	return checkerctrl.Rate{
		Times: r.GetTimes(),
		Per:   r.GetPer().AsDuration(),
	}
}

func rateToPb(r *checkerctrl.Rate) *pb.Rate {
	return &pb.Rate{
		Times: r.Times,
		Per:   durationpb.New(r.Per),
	}
}

func pbToCheckerState(in *pb.ServiceState) *checkerctrl.ServiceSettings {
	sploits := make(map[string]*checkerctrl.SploitSettings, len(in.GetSploits()))
	for sploitName, sploit := range in.GetSploits() {
		sploits[sploitName] = &checkerctrl.SploitSettings{
			Rate: pbToRate(sploit.Rate),
		}
	}

	checkers := make(map[string]*checkerctrl.CheckerSettings, len(in.GetCheckers()))
	for checkerName, checker := range in.GetCheckers() {
		checkers[checkerName] = &checkerctrl.CheckerSettings{
			Check: checkerctrl.CheckerCheckSettings{
				Rate: pbToRate(checker.Check.Rate),
			},
			Get: checkerctrl.CheckerGetSettings{
				Rate: pbToRate(checker.Get.Rate),
			},
		}
	}

	serviceState := checkerctrl.ServiceSettings{
		Target:   in.GetTarget(),
		Sploits:  sploits,
		Checkers: checkers,
	}

	return &serviceState
}

func checkerStateToPB(in *checkerctrl.ServiceSettings) *pb.ServiceState {
	sploits := make(map[string]*pb.SploitState, len(in.Sploits))
	for sploitName, sploit := range in.Sploits {
		sploits[sploitName] = &pb.SploitState{
			Rate: rateToPb(&sploit.Rate),
		}
	}

	checkers := make(map[string]*pb.CheckerState)
	for checkerName, checker := range in.Checkers {
		checkers[checkerName] = &pb.CheckerState{
			Check: &pb.CheckState{
				Rate: rateToPb(&checker.Check.Rate),
			},
			Get: &pb.GetState{
				Rate: rateToPb(&checker.Get.Rate),
			},
		}
	}

	return &pb.ServiceState{
		Target:   in.Target,
		Checkers: checkers,
		Sploits:  sploits,
	}
}
