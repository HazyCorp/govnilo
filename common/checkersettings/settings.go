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
//     target: http://example.service.local:8080
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
	SuccessPoints float64 `json:"success_points" yaml:"success_points"`
	FailPenalty   float64 `json:"fail_penalty" yaml:"fail_penalty"`
	Rate          Rate    `json:"rate" yaml:"rate"`
}

type CheckerCheckSettings struct {
	SuccessPoints float64 `json:"success_points" yaml:"success_points"`
	FailPenalty   float64 `json:"fail_penalty" yaml:"fail_penalty"`
	Rate          Rate    `json:"rate" yaml:"rate"`
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ServiceSettings struct {
	Target   string                      `json:"target" yaml:"target"`
	Checkers map[string]*CheckerSettings `json:"checkers" yaml:"checkers"`
	Sploits  map[string]*SploitSettings  `json:"sploits" yaml:"sploits"`
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
		Target:   s.Target,
		Checkers: maps.Clone(s.Checkers),
		Sploits:  maps.Clone(s.Sploits),
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

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// TODO: use configs
const TARGET_POINTS_PER_SECOND = 1000

func (s *Settings) NormalizePoints() {
	for _, svcSettings := range s.Services {
		svcSettings.normalizePointsInService(TARGET_POINTS_PER_SECOND)
	}
}

// normalizePointsInService normalizes points within one service (but all checkers).
// if sum of success points is zero, does nothing to success points.
// if sum of fail penalties is zero, does nothing to fail penalties.
func (s *ServiceSettings) normalizePointsInService(targetSum float64) {
	var pointsSum float64 = 0
	var penaltySum float64 = 0

	for _, checkerSettings := range s.Checkers {
		checksPerSecond := float64(checkerSettings.Check.Rate.Times) / checkerSettings.Check.Rate.Per.AsDuration().Seconds()
		pointsSum += checkerSettings.Check.SuccessPoints * checksPerSecond
		penaltySum += checkerSettings.Check.FailPenalty * checksPerSecond

		getsPerSecond := float64(checkerSettings.Get.Rate.Times) / checkerSettings.Get.Rate.Per.AsDuration().Seconds()
		pointsSum += checkerSettings.Get.SuccessPoints * getsPerSecond
		penaltySum += checkerSettings.Get.FailPenalty * getsPerSecond
	}

	// if sum is zero, than all the components are zero
	// if all the components are zero, we may multiply them using any number
	// (we are using zero btw)
	// thats why coefficient is counted like that

	var pointsCoefficient float64 = 0
	if pointsSum != 0.0 {
		pointsCoefficient = targetSum / pointsSum
	}

	var penaltyCoefficient float64 = 0
	if penaltySum != 0.0 {
		penaltyCoefficient = targetSum / penaltySum
	}

	for _, checkerSettings := range s.Checkers {
		checkerSettings.Check.SuccessPoints *= pointsCoefficient
		checkerSettings.Check.FailPenalty *= penaltyCoefficient

		checkerSettings.Get.SuccessPoints *= pointsCoefficient
		checkerSettings.Get.FailPenalty *= penaltyCoefficient
	}
}
