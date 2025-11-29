package settingsprovider

import (
	"testing"
	"time"

	"github.com/HazyCorp/govnilo/proto"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestValidateSettings_Valid(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			{
				Name:   "example",
				Target: "http://example.com:8080",
				Checkers: []*proto.CheckerSettings{
					{
						Name:          "checker1",
						SuccessPoints: 1.0,
						FailPenalty:   5.0,
						RunOptions: &proto.RunOptions{
							Rate: &proto.Rate{
								Times: 10,
								Per:   durationpb.New(time.Second),
							},
							MaxGoroutines: 100,
						},
					},
				},
			},
		},
	}

	err := Validate(settings)
	require.NoError(t, err)
}

func TestValidateSettings_NilSettings(t *testing.T) {
	err := Validate(nil)
	require.NoError(t, err)
}

func TestValidateSettings_NilService(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			nil,
			{
				Name:   "example",
				Target: "http://example.com:8080",
			},
		},
	}

	err := Validate(settings)
	require.Error(t, err)
}

func TestValidateSettings_EmptyServiceName(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			{
				Name:   "",
				Target: "http://example.com:8080",
			},
		},
	}

	err := Validate(settings)
	require.Error(t, err)
}

func TestValidateSettings_DuplicateServiceName(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			{
				Name:   "example",
				Target: "http://example.com:8080",
			},
			{
				Name:   "example",
				Target: "http://example2.com:8080",
			},
		},
	}

	err := Validate(settings)
	require.Error(t, err)
}

func TestValidateSettings_NilChecker(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			{
				Name:   "example",
				Target: "http://example.com:8080",
				Checkers: []*proto.CheckerSettings{
					nil,
					{
						Name:          "checker1",
						SuccessPoints: 1.0,
						FailPenalty:   5.0,
						RunOptions: &proto.RunOptions{
							Rate: &proto.Rate{
								Times: 10,
								Per:   durationpb.New(time.Second),
							},
							MaxGoroutines: 100,
						},
					},
				},
			},
		},
	}

	err := Validate(settings)
	require.Error(t, err)
}

func TestValidateSettings_EmptyCheckerName(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			{
				Name:   "example",
				Target: "http://example.com:8080",
				Checkers: []*proto.CheckerSettings{
					{
						Name:          "",
						SuccessPoints: 1.0,
						FailPenalty:   5.0,
						RunOptions: &proto.RunOptions{
							Rate: &proto.Rate{
								Times: 10,
								Per:   durationpb.New(time.Second),
							},
							MaxGoroutines: 100,
						},
					},
				},
			},
		},
	}

	err := Validate(settings)
	require.Error(t, err)
}

func TestValidateSettings_DuplicateCheckerName(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			{
				Name:   "example",
				Target: "http://example.com:8080",
				Checkers: []*proto.CheckerSettings{
					{
						Name:          "checker1",
						SuccessPoints: 1.0,
						FailPenalty:   5.0,
						RunOptions: &proto.RunOptions{
							Rate: &proto.Rate{
								Times: 10,
								Per:   durationpb.New(time.Second),
							},
							MaxGoroutines: 100,
						},
					},
					{
						Name:          "checker1",
						SuccessPoints: 2.0,
						FailPenalty:   10.0,
						RunOptions: &proto.RunOptions{
							Rate: &proto.Rate{
								Times: 20,
								Per:   durationpb.New(time.Second * 2),
							},
							MaxGoroutines: 200,
						},
					},
				},
			},
		},
	}

	err := Validate(settings)
	require.Error(t, err)
}

func TestValidateSettings_NilRunOptions(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			{
				Name:   "example",
				Target: "http://example.com:8080",
				Checkers: []*proto.CheckerSettings{
					{
						Name:          "checker1",
						SuccessPoints: 1.0,
						FailPenalty:   5.0,
						RunOptions:    nil,
					},
				},
			},
		},
	}

	err := Validate(settings)
	require.Error(t, err)
}

func TestValidateSettings_NilRate(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			{
				Name:   "example",
				Target: "http://example.com:8080",
				Checkers: []*proto.CheckerSettings{
					{
						Name:          "checker1",
						SuccessPoints: 1.0,
						FailPenalty:   5.0,
						RunOptions: &proto.RunOptions{
							Rate:          nil,
							MaxGoroutines: 100,
						},
					},
				},
			},
		},
	}

	err := Validate(settings)
	require.Error(t, err)
}

func TestValidateSettings_NilDuration(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			{
				Name:   "example",
				Target: "http://example.com:8080",
				Checkers: []*proto.CheckerSettings{
					{
						Name:          "checker1",
						SuccessPoints: 1.0,
						FailPenalty:   5.0,
						RunOptions: &proto.RunOptions{
							Rate: &proto.Rate{
								Times: 10,
								Per:   nil,
							},
							MaxGoroutines: 100,
						},
					},
				},
			},
		},
	}

	err := Validate(settings)
	require.Error(t, err)
}

func TestValidateSettings_ZeroDuration(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			{
				Name:   "example",
				Target: "http://example.com:8080",
				Checkers: []*proto.CheckerSettings{
					{
						Name:          "checker1",
						SuccessPoints: 1.0,
						FailPenalty:   5.0,
						RunOptions: &proto.RunOptions{
							Rate: &proto.Rate{
								Times: 10,
								Per:   durationpb.New(0),
							},
							MaxGoroutines: 100,
						},
					},
				},
			},
		},
	}

	err := Validate(settings)
	require.Error(t, err)
}

func TestValidateSettings_MultipleServices(t *testing.T) {
	settings := &proto.Settings{
		Services: []*proto.ServiceSettings{
			{
				Name:   "service1",
				Target: "http://service1.com:8080",
				Checkers: []*proto.CheckerSettings{
					{
						Name:          "checker1",
						SuccessPoints: 1.0,
						FailPenalty:   5.0,
						RunOptions: &proto.RunOptions{
							Rate: &proto.Rate{
								Times: 10,
								Per:   durationpb.New(time.Second),
							},
							MaxGoroutines: 100,
						},
					},
				},
			},
			{
				Name:   "service2",
				Target: "http://service2.com:8080",
				Checkers: []*proto.CheckerSettings{
					{
						Name:          "checker2",
						SuccessPoints: 2.0,
						FailPenalty:   10.0,
						RunOptions: &proto.RunOptions{
							Rate: &proto.Rate{
								Times: 20,
								Per:   durationpb.New(time.Minute),
							},
							MaxGoroutines: 200,
						},
					},
				},
			},
		},
	}

	err := Validate(settings)
	require.NoError(t, err)
}
