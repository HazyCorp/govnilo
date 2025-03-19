package checkerctrl

import (
	"context"
	"github.com/HazyCorp/govnilo/common/checkersettings"
	hazycheck2 "github.com/HazyCorp/govnilo/hazycheck"

	"github.com/pkg/errors"
	"go.uber.org/fx"
)

type SettingsProvider interface {
	GetSettings(ctx context.Context) (*checkersettings.Settings, error)
}

func validateSettings(
	converted *checkersettings.Settings,
	knownCheckers map[hazycheck2.CheckerID]hazycheck2.Checker,
	knownSploits map[hazycheck2.SploitID]hazycheck2.Sploit,
) error {
	for svcName, svcState := range converted.Services {
		for checkerName := range svcState.Checkers {
			checkerID := hazycheck2.CheckerID{
				Service: svcName,
				Name:    checkerName,
			}

			if _, exists := knownCheckers[checkerID]; !exists {
				return errors.Errorf("checker %+v not registered, config is invalid", checkerID)
			}
		}

		for sploitName := range svcState.Sploits {
			sploitID := hazycheck2.SploitID{
				Service: svcName,
				Name:    sploitName,
			}

			if _, exists := knownSploits[sploitID]; !exists {
				return errors.Errorf("sploit %+v not registered, config is invalid", sploitID)
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
