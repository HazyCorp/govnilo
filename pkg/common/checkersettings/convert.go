package checkersettings

import (
	"time"

	"github.com/HazyCorp/govnilo/proto"

	"google.golang.org/protobuf/types/known/durationpb"
)

func FromPB(s *proto.Settings) *Settings {
	settings := pbToSettings(s)
	return &settings
}

func (s *Settings) ToPB() *proto.Settings {
	return settingsToPb(s)
}

func pbToSettings(in *proto.Settings) Settings {
	services := make(map[string]*ServiceSettings, len(in.Services))
	for name, state := range in.GetServices() {
		serviceState := pbToCheckerSettings(state)
		services[name] = serviceState
	}

	return Settings{
		Services: services,
	}
}

func settingsToPb(in *Settings) *proto.Settings {
	services := make(map[string]*proto.ServiceSettings, len(in.Services))
	for name, state := range in.Services {
		pbState := checkerSettingsToPB(state)
		services[name] = pbState
	}

	return &proto.Settings{
		Services: services,
	}
}

func pbToRate(r *proto.Rate) Rate {
	return Rate{
		Times: r.GetTimes(),
		Per:   Duration(r.GetPer().AsDuration()),
	}
}

func rateToPb(r *Rate) *proto.Rate {
	return &proto.Rate{
		Times: r.Times,
		Per:   durationpb.New(time.Duration(r.Per)),
	}
}

func pbToCheckerSettings(in *proto.ServiceSettings) *ServiceSettings {
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
		Target:   in.GetTarget(),
		Sploits:  sploits,
		Checkers: checkers,
	}

	return &serviceState
}

func checkerSettingsToPB(in *ServiceSettings) *proto.ServiceSettings {
	sploits := make(map[string]*proto.SploitSettings, len(in.Sploits))
	for sploitName, sploit := range in.Sploits {
		sploits[sploitName] = &proto.SploitSettings{
			Rate: rateToPb(&sploit.Rate),
		}
	}

	checkers := make(map[string]*proto.CheckerSettings)
	for checkerName, checker := range in.Checkers {
		checkers[checkerName] = &proto.CheckerSettings{
			Check: &proto.CheckSettings{
				Rate:          rateToPb(&checker.Check.Rate),
				SuccessPoints: checker.Check.SuccessPoints,
				FailPenalty:   checker.Check.FailPenalty,
			},
			Get: &proto.GetSettings{
				Rate:          rateToPb(&checker.Get.Rate),
				SuccessPoints: checker.Get.SuccessPoints,
				FailPenalty:   checker.Get.FailPenalty,
			},
		}
	}

	return &proto.ServiceSettings{
		Target:   in.Target,
		Checkers: checkers,
		Sploits:  sploits,
	}
}
