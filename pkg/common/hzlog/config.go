package hzlog

import "log/slog"

type SkipInfraFilter struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	OnLevel string `json:"on_level" yaml:"on_level"`
	onLevel slog.Level
}

type LoggingFilter struct {
	SkipInfra SkipInfraFilter `json:"skip_infra" yaml:"skip_infra"`
}

type Config struct {
	Level  string        `json:"level" yaml:"level"`
	Mode   string        `json:"mode" yaml:"mode"`
	Filter LoggingFilter `json:"filter" yaml:"filter"`
}

func DefaultConfig() Config {
	return Config{
		Level: "debug",
		Mode:  "json",
		Filter: LoggingFilter{
			SkipInfra: SkipInfraFilter{
				Enabled: true,
				OnLevel: slog.LevelDebug.String(),
				onLevel: slog.LevelDebug,
			},
		},
	}
}
