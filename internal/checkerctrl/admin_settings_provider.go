package checkerctrl

import (
	"context"
	"log/slog"

	"github.com/HazyCorp/govnilo/internal/hazycheck"
	"github.com/HazyCorp/govnilo/pkg/adminka/adminklient"
	"github.com/HazyCorp/govnilo/pkg/common/checkersettings"

	"github.com/pkg/errors"
	"go.uber.org/fx"
)

type AdminSettingsProviderConfig struct {
	adminklient.ClientConfig `yaml:",inline"`
}

type AdminSettingsProviderIn struct {
	fx.In

	Checkers []hazycheck.Checker `group:"checkers"`
	Sploits  []hazycheck.Sploit  `group:"sploits"`
	Logger   *slog.Logger
	Config   *AdminSettingsProviderConfig
}

var _ SettingsProvider = &AdminSettingsProvider{}

type AdminSettingsProvider struct {
	client   adminklient.Client
	l        *slog.Logger
	checkers map[hazycheck.CheckerID]hazycheck.Checker
	sploits  map[hazycheck.SploitID]hazycheck.Sploit
}

func NewAdminSettingsProvider(in AdminSettingsProviderIn) (*AdminSettingsProvider, error) {
	client, err := adminklient.New(in.Config.ClientConfig, adminklient.WithLogger(in.Logger))
	if err != nil {
		return nil, errors.Wrap(err, "cannot create adminklient.Client")
	}

	checkers := make(map[hazycheck.CheckerID]hazycheck.Checker, len(in.Checkers))
	for _, c := range in.Checkers {
		checkers[c.CheckerID()] = c
	}

	sploits := make(map[hazycheck.SploitID]hazycheck.Sploit)
	for _, s := range in.Sploits {
		sploits[s.SploitID()] = s
	}

	return &AdminSettingsProvider{
		client:   client,
		l:        in.Logger.With(slog.String("component", "infra:adminka_settings_provider")),
		checkers: checkers,
		sploits:  sploits,
	}, nil
}

func (p *AdminSettingsProvider) GetSettings(ctx context.Context) (*checkersettings.Settings, error) {
	settings, err := p.client.GetConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "cannot pull settings from admin service")
	}

	if err := validateSettings(settings); err != nil {
		return nil, errors.Wrap(err, "successfully read the settings, but they are not valid")
	}

	return settings, nil
}
