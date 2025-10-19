package hzlog

import "log/slog"

type InfraFilter struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	Level   string `json:"level" yaml:"level"`
	level   slog.Level
}

type LoggingFilter struct {
	Infra InfraFilter `json:"infra" yaml:"infra"`
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
			Infra: InfraFilter{
				Enabled: true,
				Level:   slog.LevelDebug.String(),
				level:   slog.LevelDebug,
			},
		},
	}
}
