package checkerctrl

import (
	"fmt"

	hazycheck2 "github.com/HazyCorp/govnilo/internal/hazycheck"

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

// TODO: remove syncronization, all services are known at the compile time,
// no need to register metrics at runtime

func (c *Controller) checkerMetricsFor(checkerID hazycheck2.CheckerID) *checkerMetrics {
	// hot path, metrics are already registered, need only to return them
	// multiple goroutines may try to get metrics, so read lock is used
	c.checkerMetricsMu.RLock()
	m, exists := c.checkerMetrics[checkerID]
	if exists {
		c.checkerMetricsMu.RUnlock()
		return m
	}

	c.checkerMetricsMu.RUnlock()

	// metrics are not registered, need to register and save them
	// need to write to map, so we need exclusive lock
	c.checkerMetricsMu.Lock()
	defer c.checkerMetricsMu.Unlock()

	// multiple goroutines may be here, so need to check are metrics registered again
	m, exists = c.checkerMetrics[checkerID]
	if exists {
		return m
	}

	template := `%s{status=%q, method=%q, service=%q, checker=%q}`
	genMetricName := func(name, status, method string) string {
		return fmt.Sprintf(template, name, status, method, checkerID.Service, checkerID.Name)
	}

	internalErrTemplate := `%s{method=%q, service=%q, checker=%q}`
	genInternalErrMetricName := func(name, method string) string {
		return fmt.Sprintf(internalErrTemplate, name, method, checkerID.Service, checkerID.Name)
	}

	m = &checkerMetrics{
		SuccessCheckCounter: metrics.NewCounter(genMetricName("checker_runs_total", "success", "check")),
		SuccessCheckPoints:  metrics.NewFloatCounter(genMetricName("checker_points", "success", "check")),
		FailCheckCounter:    metrics.NewCounter(genMetricName("checker_runs_total", "fail", "check")),
		FailCheckPenalty:    metrics.NewFloatCounter(genMetricName("checker_points", "fail", "check")),

		CheckInternalErrorsTotal:    metrics.NewCounter(genInternalErrMetricName("checker_internal_errors_total", "check")),
		CheckInternalErrorsDuration: metrics.NewHistogram(genInternalErrMetricName("checker_internal_errors_duration", "check")),

		SuccessCheckDuration: metrics.NewHistogram(genMetricName("checker_run_duration", "success", "check")),
		FailCheckDuration:    metrics.NewHistogram(genMetricName("checker_run_duration", "fail", "check")),
	}

	c.checkerMetrics[checkerID] = m

	return m
}
