package checkersettings

import (
	"github.com/HazyCorp/govnilo/common/pb"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
)

func FromPB(s *pb.Settings) *Settings {
	settings := pbToSettings(s)
	return &settings
}

func (s *Settings) ToPB() *pb.Settings {
	return settingsToPb(s)
}

func pbToSettings(in *pb.Settings) Settings {
	services := make(map[string]*ServiceSettings, len(in.Services))
	for name, state := range in.GetServices() {
		serviceState := pbToCheckerSettings(state)
		services[name] = serviceState
	}

	return Settings{
		Services: services,
	}
}

func settingsToPb(in *Settings) *pb.Settings {
	services := make(map[string]*pb.ServiceSettings, len(in.Services))
	for name, state := range in.Services {
		pbState := checkerSettingsToPB(state)
		services[name] = pbState
	}

	return &pb.Settings{
		Services: services,
	}
}

func pbToRate(r *pb.Rate) Rate {
	return Rate{
		Times: r.GetTimes(),
		Per:   Duration(r.GetPer().AsDuration()),
	}
}

func rateToPb(r *Rate) *pb.Rate {
	return &pb.Rate{
		Times: r.Times,
		Per:   durationpb.New(time.Duration(r.Per)),
	}
}

func pbToCheckerSettings(in *pb.ServiceSettings) *ServiceSettings {
	sploits := make(map[string]*SploitSettings, len(in.GetSploits()))
	for sploitName, sploit := range in.GetSploits() {
		sploits[sploitName] = &SploitSettings{
			Rate: pbToRate(sploit.Rate),
		}
	}

	checkers := make(map[string]*CheckerSettings, len(in.GetCheckers()))
	for checkerName, checker := range in.GetCheckers() {
		checkers[checkerName] = &CheckerSettings{
			Check: CheckerCheckSettings{
				SuccessPoints: checker.Check.GetSuccessPoints(),
				FailPenalty:   checker.Check.GetFailPenalty(),
				Rate:          pbToRate(checker.Check.Rate),
			},
			Get: CheckerGetSettings{
				SuccessPoints: checker.Get.GetSuccessPoints(),
				FailPenalty:   checker.Get.GetFailPenalty(),
				Rate:          pbToRate(checker.Get.Rate),
			},
		}
	}

	serviceState := ServiceSettings{
		TargetPort: in.GetTargetPort(),
		Sploits:    sploits,
		Checkers:   checkers,
	}

	return &serviceState
}

func checkerSettingsToPB(in *ServiceSettings) *pb.ServiceSettings {
	sploits := make(map[string]*pb.SploitSettings, len(in.Sploits))
	for sploitName, sploit := range in.Sploits {
		sploits[sploitName] = &pb.SploitSettings{
			Rate: rateToPb(&sploit.Rate),
		}
	}

	checkers := make(map[string]*pb.CheckerSettings)
	for checkerName, checker := range in.Checkers {
		checkers[checkerName] = &pb.CheckerSettings{
			Check: &pb.CheckSettings{
				Rate:          rateToPb(&checker.Check.Rate),
				SuccessPoints: checker.Check.SuccessPoints,
				FailPenalty:   checker.Check.FailPenalty,
			},
			Get: &pb.GetSettings{
				Rate:          rateToPb(&checker.Get.Rate),
				SuccessPoints: checker.Get.SuccessPoints,
				FailPenalty:   checker.Get.FailPenalty,
			},
		}
	}

	return &pb.ServiceSettings{
		TargetPort: in.TargetPort,
		Checkers:   checkers,
		Sploits:    sploits,
	}
}
