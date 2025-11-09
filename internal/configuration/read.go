package configuration

import (
	"os"
	"time"

	"github.com/HazyCorp/govnilo/internal/checkerctrl"
	"github.com/HazyCorp/govnilo/internal/cmd/globflags"
	"github.com/HazyCorp/govnilo/internal/metricsrv"
	"github.com/HazyCorp/govnilo/pkg/common/hzlog"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"gopkg.in/yaml.v3"
)

type Config struct {
	fx.Out

	Logging          hzlog.Config                       `json:"logging" yaml:"logging"`
	Serve            Serve                              `json:"serve" yaml:"serve"`
	AsyncFileStore   checkerctrl.AsyncFileStoreConfig   `json:"async_file_store" yaml:"async_file_store"`
	Controller       checkerctrl.Config                 `json:"controller" yaml:"controller"`
	Metrics          metricsrv.Config                   `json:"metrics" yaml:"metrics"`
	SettingsProvider checkerctrl.SettingsProviderConfig `json:"settings_provider" yaml:"settings_provider"`
	Redis            Redis                              `json:"redis" yaml:"redis"`
}

func defaultConfig() *Config {
	return &Config{
		Serve: Serve{
			Port: 13337,
		},
		AsyncFileStore: checkerctrl.AsyncFileStoreConfig{
			Path:         "/tmp/checker_state.bin",
			SyncInterval: time.Second,
		},
		Controller: checkerctrl.Config{
			SyncInterval: time.Second,
		},
		Metrics: metricsrv.Config{
			Port: 14448,
		},
		SettingsProvider: checkerctrl.SettingsProviderConfig{
			FromFile: &checkerctrl.FileSettingsProviderConfig{
				Path: "/etc/govnilo/settings.yaml",
			},
		},
		Logging: hzlog.DefaultConfig(),
		Redis: Redis{
			Host:     "localhost",
			Port:     6379,
			Username: "govnilo",
			Password: "govnilo-rulit",
			DB:       0,
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

	data = []byte(os.ExpandEnv(string(data)))

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

	if c.Redis.Validate() != nil {
		return errors.Wrap(c.Redis.Validate(), "invalid redis config")
	}

	return nil
}
