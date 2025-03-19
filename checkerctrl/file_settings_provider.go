package checkerctrl

import (
	"context"
	"os"

	"github.com/HazyCorp/govnilo/common/checkersettings"
	"github.com/HazyCorp/govnilo/hazycheck"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"gopkg.in/yaml.v3"
)

type FileSettingsProviderIn struct {
	fx.In

	Checkers []hazycheck.Checker `group:"checkers"`
	Sploits  []hazycheck.Sploit  `group:"sploits"`
	Config   *FileSettingsProviderConfig
}

type FileSettingsProviderConfig struct {
	Path string
}

var _ SettingsProvider = &FileSettingsProvider{}

type FileSettingsProvider struct {
	c        *FileSettingsProviderConfig
	checkers map[hazycheck.CheckerID]hazycheck.Checker
	sploits  map[hazycheck.SploitID]hazycheck.Sploit
}

func NewFileSettingsProvider(in FileSettingsProviderIn) *FileSettingsProvider {
	checkers := make(map[hazycheck.CheckerID]hazycheck.Checker)
	for _, c := range in.Checkers {
		checkers[c.CheckerID()] = c
	}

	sploits := make(map[hazycheck.SploitID]hazycheck.Sploit)
	for _, s := range in.Sploits {
		sploits[s.SploitID()] = s
	}

	return &FileSettingsProvider{
		checkers: checkers,
		sploits:  sploits,
		c:        in.Config,
	}
}

func (p *FileSettingsProvider) GetSettings(ctx context.Context) (*checkersettings.Settings, error) {
	data, err := os.ReadFile(p.c.Path)
	if err != nil {
		return nil, errors.Wrap(err, "cannot read data from file")
	}

	var s checkersettings.Settings
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, errors.Wrap(err, "cannot parse the config as yaml")
	}

	if err := validateSettings(&s, p.checkers, p.sploits); err != nil {
		return nil, errors.Wrap(err, "read the settings from file, but they were not valid")
	}

	return &s, nil
}
