package checkerserver

import (
	"context"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/HazyCorp/govnilo/internal/checkerctrl"
	"github.com/HazyCorp/govnilo/pkg/hazycheck"
	"github.com/HazyCorp/govnilo/pkg/pb"
)

type CheckerServer struct {
	pb.UnimplementedCheckerServiceServer

	storage  checkerctrl.ControllerStorage
	checkers map[string]hazycheck.Checker
	l        *zap.Logger
}

type CheckerServerIn struct {
	fx.In

	Storage  checkerctrl.ControllerStorage
	Checkers []hazycheck.Checker `group:"checkers"`
	Logger   *zap.Logger
}

func New(in CheckerServerIn) *CheckerServer {
	checkers := make(map[string]hazycheck.Checker)
	for _, c := range in.Checkers {
		checkers[c.ServiceName()] = c
	}

	return &CheckerServer{
		storage:  in.Storage,
		checkers: checkers,
		l:        in.Logger,
	}
}

func NewFX(in CheckerServerIn, srv *grpc.Server) *CheckerServer {
	s := New(in)
	pb.RegisterCheckerServiceServer(srv, s)
	reflection.Register(srv)

	return s
}

func (s *CheckerServer) SetCheckerState(
	ctx context.Context,
	in *pb.SetCheckerStateReq,
) (*pb.SetCheckerStateRsp, error) {
	converted := pbToState(in.GetState())
	for svcName := range converted.Services {
		if _, exists := s.checkers[svcName]; !exists {
			return nil, errors.Errorf("checker %q not registered, config is invalid", svcName)
		}
	}

	if err := s.storage.SetContestState(ctx, &converted); err != nil {
		return nil, errors.Wrap(err, "cannot save state in storage")
	}

	return &pb.SetCheckerStateRsp{}, nil
}

func (s *CheckerServer) GetCheckerState(
	ctx context.Context,
	in *pb.GetCheckerStateReq,
) (*pb.GetCheckerStateRsp, error) {
	state, err := s.storage.GetContestState(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot retrieve state from storage")
	}

	return &pb.GetCheckerStateRsp{State: stateToPb(state)}, nil
}

func (s *CheckerServer) GetSLA(ctx context.Context, req *pb.GetSLAReq) (*pb.GetSLAResponse, error) {
	slas := make(map[string]*pb.SLA, len(s.checkers))
	for svcName := range s.checkers {
		sla, err := s.storage.GetServiceSLA(ctx, svcName)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot get sla for service %s from storage", svcName)
		}

		slas[svcName] = &pb.SLA{
			SuccessCount: uint64(sla.SuccessfullAttempts),
			TotalCount:   uint64(sla.TotalAttempts),
		}
	}

	return &pb.GetSLAResponse{
		Slas: slas,
	}, nil
}

func pbToState(in *pb.CheckerState) checkerctrl.ServicesContestState {
	services := make(map[string]checkerctrl.ServiceCheckerState, len(in.Services))
	for name, state := range in.GetServices() {
		serviceState := pbToServiceCheckerState(state)
		services[name] = *serviceState
	}

	return checkerctrl.ServicesContestState{
		Services: services,
	}
}

func stateToPb(in *checkerctrl.ServicesContestState) *pb.CheckerState {
	services := make(map[string]*pb.ServiceState, len(in.Services))
	for name, state := range in.Services {
		pbState := serviceCheckerStateToPb(&state)
		services[name] = pbState
	}

	return &pb.CheckerState{
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

func pbToServiceCheckerState(in *pb.ServiceState) *checkerctrl.ServiceCheckerState {
	sploits := make(map[string]checkerctrl.SploitState, len(in.GetSploitsState()))
	for sploitName, sploit := range in.GetSploitsState() {
		sploits[sploitName] = checkerctrl.SploitState{
			Rate: pbToRate(sploit.Rate),
		}
	}

	serviceState := checkerctrl.ServiceCheckerState{
		Target:    in.GetTarget(),
		CheckRate: pbToRate(in.GetCheckState().GetCheckerCheckRate()),
		GetRate:   pbToRate(in.GetCheckState().GetCheckerGetRate()),
		Sploits:   sploits,
	}

	return &serviceState
}

func serviceCheckerStateToPb(in *checkerctrl.ServiceCheckerState) *pb.ServiceState {
	sploits := make(map[string]*pb.SploitState)
	for sploitName, sploit := range in.Sploits {
		sploits[sploitName] = &pb.SploitState{
			Rate: rateToPb(&sploit.Rate),
		}
	}

	return &pb.ServiceState{
		Target: in.Target,
		CheckState: &pb.CheckState{
			CheckerCheckRate: rateToPb(&in.CheckRate),
			CheckerGetRate:   rateToPb(&in.GetRate),
		},
		SploitsState: sploits,
	}
}
