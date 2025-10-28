package configuration

import "errors"

type Redis struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
	DB       int    `json:"db" yaml:"db"`
}

func (r *Redis) Validate() error {
	if r.Host == "" {
		return errors.New("redis host is required")
	}
	if r.Port == 0 {
		return errors.New("redis port is required")
	}
	if r.Username == "" {
		return errors.New("redis username is required")
	}
	if r.Password == "" {
		return errors.New("redis password is required")
	}
	return nil
}
