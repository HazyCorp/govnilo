package checkerctrl

import (
	"fmt"

	"github.com/HazyCorp/govnilo/internal/hazycheck"

	"github.com/VictoriaMetrics/metrics"
)

type checkerMetrics struct {
	SuccessCheckCounter *metrics.Counter
	SuccessCheckPoints  *metrics.FloatCounter
	FailCheckCounter    *metrics.Counter
	FailCheckPenalty    *metrics.FloatCounter

	CheckInternalErrorsTotal    *metrics.Counter
	CheckInternalErrorsDuration *metrics.Histogram

	SuccessCheckDuration *metrics.Histogram
	FailCheckDuration    *metrics.Histogram
}

func (c *Controller) metricsFor(checkerID hazycheck.CheckerID) *checkerMetrics {
	template := `%s{status=%q, method=%q, service=%q, checker=%q}`
	genMetricName := func(name, status, method string) string {
		return fmt.Sprintf(template, name, status, method, checkerID.Service, checkerID.Name)
	}

	internalErrTemplate := `%s{method=%q, service=%q, checker=%q}`
	genInternalErrMetricName := func(name, method string) string {
		return fmt.Sprintf(internalErrTemplate, name, method, checkerID.Service, checkerID.Name)
	}

	return &checkerMetrics{
		SuccessCheckCounter: metrics.NewCounter(genMetricName("checker_runs_total", "success", "check")),
		SuccessCheckPoints:  metrics.NewFloatCounter(genMetricName("checker_points", "success", "check")),
		FailCheckCounter:    metrics.NewCounter(genMetricName("checker_runs_total", "fail", "check")),
		FailCheckPenalty:    metrics.NewFloatCounter(genMetricName("checker_points", "fail", "check")),

		CheckInternalErrorsTotal:    metrics.NewCounter(genInternalErrMetricName("checker_internal_errors_total", "check")),
		CheckInternalErrorsDuration: metrics.NewHistogram(genInternalErrMetricName("checker_internal_errors_duration", "check")),

		SuccessCheckDuration: metrics.NewHistogram(genMetricName("checker_run_duration", "success", "check")),
		FailCheckDuration:    metrics.NewHistogram(genMetricName("checker_run_duration", "fail", "check")),
	}
}

func (c *Controller) registerMetrics(checkerID hazycheck.CheckerID) {
	c.checkerMetrics[checkerID] = c.metricsFor(checkerID)
}

func (c *Controller) checkerMetricsFor(checkerID hazycheck.CheckerID) *checkerMetrics {
	m, exists := c.checkerMetrics[checkerID]
	if !exists {
		panic(fmt.Sprintf("trying to fetch metrics of not created checker %+v", checkerID))
	}

	return m
}
