package hzlog

type Config struct {
	Level string `json:"level" yaml:"level"`
	Mode  string `json:"mode" yaml:"mode"`
}

func DefaultConfig() Config {
	return Config{
		Level: "debug",
		Mode:  "json",
	}
}
