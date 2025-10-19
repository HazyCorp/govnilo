package checkersettings

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSimple(t *testing.T) {
	settings := Settings{
		Services: map[string]*ServiceSettings{
			"example": {
				Checkers: map[string]*CheckerSettings{
					"abacaba": {
						Check: CheckerCheckSettings{
							SuccessPoints: 100,
							FailPenalty:   200,
							RunOptions: RunOptions{
								Rate: Rate{Times: 1, Per: Duration(time.Second)},
							},
						},
						Get: CheckerGetSettings{
							SuccessPoints: 100,
							FailPenalty:   200,
							RunOptions: RunOptions{
								Rate: Rate{Times: 1, Per: Duration(time.Second)},
							},
						},
					},
				},
			},
		},
	}

	settings.NormalizePoints()

	require.Equal(t, float64(500), settings.Services["example"].Checkers["abacaba"].Check.SuccessPoints)
	require.Equal(t, float64(500), settings.Services["example"].Checkers["abacaba"].Get.SuccessPoints)

	require.Equal(t, float64(500), settings.Services["example"].Checkers["abacaba"].Get.FailPenalty)
	require.Equal(t, float64(500), settings.Services["example"].Checkers["abacaba"].Get.FailPenalty)
}

func TestNormalize_RateIsCounted(t *testing.T) {
	settings := Settings{
		Services: map[string]*ServiceSettings{
			"example": {
				Checkers: map[string]*CheckerSettings{
					"abacaba": {
						Check: CheckerCheckSettings{
							SuccessPoints: 100,
							FailPenalty:   200,
							RunOptions: RunOptions{
								Rate: Rate{Times: 10, Per: Duration(time.Second)},
							},
						},
						Get: CheckerGetSettings{
							SuccessPoints: 100,
							FailPenalty:   200,
							RunOptions: RunOptions{
								Rate: Rate{Times: 1, Per: Duration(time.Second)},
							},
						},
					},
				},
			},
		},
	}

	settings.NormalizePoints()

	s := settings.Services["example"].Checkers["abacaba"]
	getSettings := s.Get
	checkSettings := s.Check

	getPerSecond := float64(getSettings.RunOptions.Rate.Times) / getSettings.RunOptions.Rate.Per.AsDuration().Seconds()
	checkPerSecond := float64(checkSettings.RunOptions.Rate.Times) / checkSettings.RunOptions.Rate.Per.AsDuration().Seconds()

	checkSuccessPerSecond := checkSettings.SuccessPoints * checkPerSecond
	getSuccessPerSecond := getSettings.SuccessPoints * getPerSecond

	// we need to save the relation between checker costs
	require.Equal(t, float64(10), checkSuccessPerSecond/getSuccessPerSecond)
	// we need sum to be equal to targetsum
	require.Equal(t, float64(1000), checkSuccessPerSecond+getSuccessPerSecond)

	checkPenaltyPerSecond := checkSettings.FailPenalty * checkPerSecond
	getPenaltyPerSecond := getSettings.FailPenalty * getPerSecond

	require.Equal(t, float64(10), checkPenaltyPerSecond/getPenaltyPerSecond)
	require.Equal(t, float64(1000), checkPenaltyPerSecond+getPenaltyPerSecond)
}
