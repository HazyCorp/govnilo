package checkerctrl

import (
	"context"

	"github.com/HazyCorp/govnilo/proto"
	"github.com/pkg/errors"

	"go.uber.org/fx"
)

type SettingsProvider interface {
	GetSettings(ctx context.Context) (*proto.Settings, error)
}

func validateSettings(settings *proto.Settings) error {
	seenServices := make(map[string]bool)

	for svcIdx, svc := range settings.GetServices() {
		if svc == nil {
			return errors.Errorf("services[%d]: service description cannot be nil", svcIdx)
		}

		svcName := svc.GetName()
		if svcName == "" {
			return errors.Errorf("services[%d]: service name cannot be empty", svcIdx)
		}

		// Check for duplicate service names
		if seenServices[svcName] {
			return errors.Errorf("services[%d (%s)]: duplicate service name", svcIdx, svcName)
		}
		seenServices[svcName] = true

		seenCheckers := make(map[string]bool)
		for checkerIdx, checkerDesc := range svc.GetCheckers() {
			if checkerDesc == nil {
				return errors.Errorf("services[%d (%s)].checkers[%d]: checker description cannot be nil", svcIdx, svcName, checkerIdx)
			}

			checkerName := checkerDesc.GetName()
			if checkerName == "" {
				return errors.Errorf("services[%d (%s)].checkers[%d]: checker name cannot be empty", svcIdx, svcName, checkerIdx)
			}

			// Check for duplicate checker names within the service
			if seenCheckers[checkerName] {
				return errors.Errorf("services[%d (%s)].checkers[%d (%s)]: duplicate checker name", svcIdx, svcName, checkerIdx, checkerName)
			}
			seenCheckers[checkerName] = true

			runOpts := checkerDesc.GetRunOptions()
			if runOpts == nil {
				return errors.Errorf("services[%d (%s)].checkers[%d (%s)].run_options cannot be nil", svcIdx, svcName, checkerIdx, checkerName)
			}

			rate := runOpts.GetRate()
			if rate == nil {
				return errors.Errorf("services[%d (%s)].checkers[%d (%s)].run_options.rate cannot be nil", svcIdx, svcName, checkerIdx, checkerName)
			}

			if rate.GetPer() == nil || rate.GetPer().AsDuration() == 0 {
				return errors.Errorf("services[%d (%s)].checkers[%d (%s)].run_options.rate.per cannot be 0", svcIdx, svcName, checkerIdx, checkerName)
			}
		}
	}

	return nil
}

type SettingsProviderConfig struct {
	fx.Out

	FromFile  *FileSettingsProviderConfig  `json:"from_file" yaml:"from_file"`
	FromAdmin *AdminSettingsProviderConfig `json:"from_admin" yaml:"from_admin"`
}
