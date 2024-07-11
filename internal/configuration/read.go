package configuration

import (
	"flag"
	"os"

	"github.com/pkg/errors"
	"go.uber.org/fx"
	"gopkg.in/yaml.v3"

	"github.com/HazyCorp/checker/internal/checkerctrl"
	"github.com/HazyCorp/checker/internal/metricsrv"
)

type Config struct {
	fx.Out

	Serve          Serve                            `json:"serve"            yaml:"serve"`
	AsyncFileStore checkerctrl.AsyncFileStoreConfig `json:"async_file_store" yaml:"async_file_store"`
	Controller     checkerctrl.Config               `json:"controller"       yaml:"controller"`
	Metrics        metricsrv.Config                 `json:"metrics"          yaml:"metrics"`
}

func Read() (Config, error) {
	var confPath string
	flag.StringVar(&confPath, "conf", "", "Path to configuration")
	flag.Parse()

	if confPath == "" {
		return Config{}, errors.Errorf("config path not provided")
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

	return nil
}
