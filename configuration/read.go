package configuration

import (
	checkerctrl2 "github.com/HazyCorp/govnilo/checkerctrl"
	"github.com/HazyCorp/govnilo/cmd/govnilo/globflags"
	"github.com/HazyCorp/govnilo/common/hzlog"
	"github.com/HazyCorp/govnilo/metricsrv"
	"gopkg.in/yaml.v3"
	"os"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/fx"
)

type Config struct {
	fx.Out

	Logging          hzlog.Config                        `json:"logging" yaml:"logging"`
	Serve            Serve                               `json:"serve" yaml:"serve"`
	AsyncFileStore   checkerctrl2.AsyncFileStoreConfig   `json:"async_file_store" yaml:"async_file_store"`
	Controller       checkerctrl2.Config                 `json:"controller" yaml:"controller"`
	Metrics          metricsrv.Config                    `json:"metrics" yaml:"metrics"`
	SettingsProvider checkerctrl2.SettingsProviderConfig `json:"settings_provider" yaml:"settings_provider"`
}

func defaultConfig() *Config {
	return &Config{
		Serve: Serve{
			Port: 13337,
		},
		AsyncFileStore: checkerctrl2.AsyncFileStoreConfig{
			Path:         "/tmp/checker_state.bin",
			SyncInterval: time.Second,
		},
		Controller: checkerctrl2.Config{
			SyncInterval: time.Second,
		},
		Metrics: metricsrv.Config{
			Port: 14448,
		},
		SettingsProvider: checkerctrl2.SettingsProviderConfig{
			FromFile: &checkerctrl2.FileSettingsProviderConfig{
				Path: "/etc/govnilo/settings.yaml",
			},
		},
	}
}

func Read() (Config, error) {
	confPath := globflags.ConfigPath

	if confPath == "" {
		return *defaultConfig(), nil
	}

	data, err := os.ReadFile(confPath)
	if err != nil {
		return Config{}, errors.Wrapf(err, "cannot read config at %s", confPath)
	}

	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, errors.Wrap(err, "cannot parse config as yaml")
	}

	if err := Validate(&c); err != nil {
		return Config{}, errors.Wrap(err, "invalid config provided")
	}

	return c, nil
}

func Validate(c *Config) error {
	if c.Serve.Port == 0 {
		return errors.Errorf("config.serve.port must be provided")
	}
	if c.AsyncFileStore.Path == "" {
		return errors.Errorf("config.async_file_store.path must be provided")
	}
	if c.Controller.SyncInterval == 0 {
		return errors.Errorf("config.controller.sync_interval must be provided")
	}
	if c.Metrics.Port == 0 {
		return errors.Errorf("config.metrics.port must be provided")
	}

	if c.SettingsProvider.FromAdmin == nil && c.SettingsProvider.FromFile == nil {
		return errors.Errorf("settings provider must be chosen")
	}
	if c.SettingsProvider.FromAdmin != nil && c.SettingsProvider.FromFile != nil {
		return errors.Errorf("only one settings provider must be chosen")
	}

	return nil
}
