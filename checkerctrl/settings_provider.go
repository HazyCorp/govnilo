package checkerctrl

import (
	"context"

	"github.com/HazyCorp/govnilo/common/checkersettings"
	"github.com/pkg/errors"

	"go.uber.org/fx"
)

type SettingsProvider interface {
	GetSettings(ctx context.Context) (*checkersettings.Settings, error)
}

func validateSettings(
	converted *checkersettings.Settings,
) error {
	// TODO: add some validations
	for svcName, svc := range converted.Services {
		if svc == nil {
			return errors.Errorf("service description cannot be nil, but service %s was", svcName)
		}

		for checkerName, checkerDesc := range svc.Checkers {
			if checkerDesc == nil {
				return errors.Errorf("checker description cannot be nil, but checker %s:%s was", svcName, checkerName)
			}
		}

		for sploitName, sploitDesc := range svc.Sploits {
			if sploitDesc == nil {
				return errors.Errorf("checker description cannot be nil, but checker %s:%s was", svcName, sploitName)
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
