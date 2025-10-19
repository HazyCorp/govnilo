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
			RunOptions: RunOptions{
				Rate:          pbToRate(sploit.RunOptions.Rate),
				MaxGoroutines: int(sploit.RunOptions.MaxGoroutines),
			},
		}
	}

	checkers := make(map[string]*CheckerSettings, len(in.GetCheckers()))
	for checkerName, checker := range in.GetCheckers() {
		checkers[checkerName] = &CheckerSettings{
			Check: CheckerCheckSettings{
				RunOptions: RunOptions{
					Rate:          pbToRate(checker.Check.RunOptions.Rate),
					MaxGoroutines: int(checker.Check.RunOptions.MaxGoroutines),
				},
				SuccessPoints: checker.Check.SuccessPoints,
				FailPenalty:   checker.Check.FailPenalty,
			},
			Get: CheckerGetSettings{
				RunOptions: RunOptions{
					Rate:          pbToRate(checker.Get.RunOptions.Rate),
					MaxGoroutines: int(checker.Get.RunOptions.MaxGoroutines),
				},
				SuccessPoints: checker.Get.SuccessPoints,
				FailPenalty:   checker.Get.FailPenalty,
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
			RunOptions: &proto.RunOptions{
				Rate:          rateToPb(&sploit.RunOptions.Rate),
				MaxGoroutines: int32(sploit.RunOptions.MaxGoroutines),
			},
		}
	}

	checkers := make(map[string]*proto.CheckerSettings)
	for checkerName, checker := range in.Checkers {
		checkers[checkerName] = &proto.CheckerSettings{
			Check: &proto.CheckSettings{
				RunOptions: &proto.RunOptions{
					Rate:          rateToPb(&checker.Check.RunOptions.Rate),
					MaxGoroutines: int32(checker.Check.RunOptions.MaxGoroutines),
				},
				SuccessPoints: checker.Check.SuccessPoints,
				FailPenalty:   checker.Check.FailPenalty,
			},
			Get: &proto.GetSettings{
				RunOptions: &proto.RunOptions{
					Rate:          rateToPb(&checker.Get.RunOptions.Rate),
					MaxGoroutines: int32(checker.Get.RunOptions.MaxGoroutines),
				},
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
