package checkersettings

import (
	"encoding/json"
	"errors"
	"maps"
	"time"
)

// example settings (represented in yaml)
// ...
// services:
//   example:
//     target_port: 8080
//     checkers:
//       simple_user_flow:
//         check:
//			 success_points: 1
//           fail_penalty: 5
//           rate:
//             times: 500
//             per: 1s
//         get:
//			 success_points: 1
//           fail_penalty: 10
//           rate:
//             times: 5
//             per: 1s
//       user_with_big_payload:
//         check:
//			 success_points: 100
//           fail_penalty: 50
//           rate:
//             times: 5
//             per: 1m
//          get:
//			  success_points: 5000
//            fail_penalty: 10000
//            rate:
//              times: 1
//              per: 1h
//     sploits:
//       drop_database_sql_injection:
//         rate:
//           times: 1
//             per: 1s

type Duration time.Duration

func (d Duration) AsDuration() time.Duration {
	return time.Duration(d)
}

func (d *Duration) MarshalText() (text []byte, err error) {
	return []byte(time.Duration(*d).String()), nil
}

func (d *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}

	*d = Duration(parsed)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.New("invalid duration")
	}
}

type Rate struct {
	Times uint64   `json:"times" yaml:"times"`
	Per   Duration `json:"per" yaml:"per"`
}

type SploitSettings struct {
	Rate Rate `json:"rate" yaml:"rate"`
}

type CheckerSettings struct {
	Check CheckerCheckSettings `json:"check" yaml:"check"`
	Get   CheckerGetSettings   `json:"get" yaml:"get"`
}

type CheckerGetSettings struct {
	SuccessPoints uint64 `json:"success_points" yaml:"success_points"`
	FailPenalty   uint64 `json:"fail_penalty" yaml:"fail_penalty"`
	Rate          Rate   `json:"rate" yaml:"rate"`
}

type CheckerCheckSettings struct {
	SuccessPoints uint64 `json:"success_points" yaml:"success_points"`
	FailPenalty   uint64 `json:"fail_penalty" yaml:"fail_penalty"`
	Rate          Rate   `json:"rate" yaml:"rate"`
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ServiceSettings struct {
	TargetPort uint32                      `json:"target_port" yaml:"target_port"`
	Checkers   map[string]*CheckerSettings `json:"checkers" yaml:"checkers"`
	Sploits    map[string]*SploitSettings  `json:"sploits" yaml:"sploits"`
}

func (s *ServiceSettings) Init() {
	s.Checkers = make(map[string]*CheckerSettings)
	s.Sploits = make(map[string]*SploitSettings)
}

func (s *ServiceSettings) Clone() ServiceSettings {
	if s == nil {
		return ServiceSettings{}
	}

	return ServiceSettings{
		TargetPort: s.TargetPort,
		Checkers:   maps.Clone(s.Checkers),
		Sploits:    maps.Clone(s.Sploits),
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type Settings struct {
	Services map[string]*ServiceSettings `yaml:"services" json:"services"`
}

func (s *Settings) Init() {
	s.Services = make(map[string]*ServiceSettings)
}

func (s *Settings) Clone() *Settings {
	if s == nil {
		return nil
	}

	services := make(map[string]*ServiceSettings, len(s.Services))
	for k, v := range s.Services {
		svc := v.Clone()
		services[k] = &svc
	}

	return &Settings{Services: services}
}
