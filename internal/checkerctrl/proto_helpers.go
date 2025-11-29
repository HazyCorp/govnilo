package checkerctrl

import (
	"time"

	"github.com/HazyCorp/govnilo/pkg/raterunner"
	"github.com/HazyCorp/govnilo/proto"
)

// serviceMap converts repeated ServiceSettings to a map keyed by service name
func serviceMap(settings *proto.Settings) map[string]*proto.ServiceSettings {
	if settings == nil {
		return nil
	}

	result := make(map[string]*proto.ServiceSettings, len(settings.GetServices()))
	for _, svc := range settings.GetServices() {
		if svc != nil {
			result[svc.GetName()] = svc
		}
	}
	return result
}

// checkersMap converts repeated CheckerSettings to a map keyed by checker name
func checkersMap(checkers []*proto.CheckerSettings) map[string]*proto.CheckerSettings {
	if checkers == nil {
		return nil
	}

	result := make(map[string]*proto.CheckerSettings, len(checkers))
	for _, checker := range checkers {
		if checker != nil {
			result[checker.GetName()] = checker
		}
	}
	return result
}

// normalizePoints normalizes points across all checkers within services
// to achieve TARGET_POINTS_PER_SECOND total points per second
const TARGET_POINTS_PER_SECOND = 1000

func normalizePoints(settings *proto.Settings) {
	if settings == nil {
		return
	}

	servicesMap := serviceMap(settings)
	for _, svcSettings := range servicesMap {
		normalizePointsInService(svcSettings, TARGET_POINTS_PER_SECOND)
	}
}

// normalizePointsInService normalizes points within one service (but all checkers).
// if sum of success points is zero, does nothing to success points.
// if sum of fail penalties is zero, does nothing to fail penalties.
func normalizePointsInService(svcSettings *proto.ServiceSettings, targetSum float64) {
	if svcSettings == nil {
		return
	}

	var pointsSum float64 = 0
	var penaltySum float64 = 0

	for _, checkerSettings := range svcSettings.GetCheckers() {
		if checkerSettings == nil {
			continue
		}

		runOpts := checkerSettings.GetRunOptions()
		if runOpts == nil {
			continue
		}

		rate := runOpts.GetRate()
		if rate == nil || rate.GetPer() == nil {
			continue
		}

		perSecond := float64(rate.GetTimes()) / rate.GetPer().AsDuration().Seconds()
		pointsSum += checkerSettings.GetSuccessPoints() * perSecond
		penaltySum += checkerSettings.GetFailPenalty() * perSecond
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

	for _, checkerSettings := range svcSettings.GetCheckers() {
		if checkerSettings == nil {
			continue
		}

		checkerSettings.SuccessPoints *= pointsCoefficient
		checkerSettings.FailPenalty *= penaltyCoefficient
	}
}

// protoRateToRaterunner converts proto Rate to raterunner.Rate
func protoRateToRaterunner(rate *proto.Rate) raterunner.Rate {
	if rate == nil {
		return raterunner.ZeroRate
	}

	var per time.Duration
	if rate.GetPer() != nil {
		per = rate.GetPer().AsDuration()
	}

	return raterunner.Rate{
		Times: rate.GetTimes(),
		Per:   per,
	}
}
