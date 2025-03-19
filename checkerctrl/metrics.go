package checkerctrl

import (
	"fmt"
	hazycheck2 "github.com/HazyCorp/govnilo/hazycheck"

	"github.com/VictoriaMetrics/metrics"
)

type checkerMetrics struct {
	SuccessCheckCounter *metrics.Counter
	SuccessCheckPoints  *metrics.Counter
	FailCheckCounter    *metrics.Counter
	FailCheckPenalty    *metrics.Counter

	SuccessGetCounter *metrics.Counter
	SuccessGetPoints  *metrics.Counter
	FailGetCounter    *metrics.Counter
	FailGetPenalty    *metrics.Counter

	SuccessCheckDuration *metrics.Histogram
	FailCheckDuration    *metrics.Histogram

	SuccessGetDuration *metrics.Histogram
	FailGetDuration    *metrics.Histogram
}

type sploitMetrics struct {
	SuccessCounter *metrics.Counter
	FailCounter    *metrics.Counter

	SuccessDuration *metrics.Histogram
	FailDuration    *metrics.Histogram
}

func (c *Controller) sploitMetricsFor(sploitID hazycheck2.SploitID) *sploitMetrics {
	// hot path, metrics are already registered, need only to return them
	// multiple goroutines may try to get metrics, so read lock is used
	c.sploitMetricsMu.RLock()
	m, exists := c.sploitMetrics[sploitID]
	if exists {
		c.sploitMetricsMu.RUnlock()
		return m
	}

	c.sploitMetricsMu.RUnlock()

	// metrics are not registered, need to register and save them
	// need to write to map, so we need exclusive lock
	c.sploitMetricsMu.Lock()
	defer c.sploitMetricsMu.Unlock()

	// multiple goroutines may be here, so need to check are metrics registered by another
	// gorutine earlier
	m, exists = c.sploitMetrics[sploitID]
	if exists {
		return m
	}

	template := `%s{status=%q, service=%q, sploit=%q}`
	m = &sploitMetrics{
		SuccessCounter: metrics.NewCounter(
			fmt.Sprintf(template, "sploit_runs_total", "success", sploitID.Service, sploitID.Name),
		),
		FailCounter: metrics.NewCounter(
			fmt.Sprintf(template, "sploit_runs_total", "fail", sploitID.Service, sploitID.Name),
		),

		SuccessDuration: metrics.NewHistogram(
			fmt.Sprintf(template, "sploit_duration", "success", sploitID.Service, sploitID.Name),
		),
		FailDuration: metrics.NewHistogram(
			fmt.Sprintf(template, "sploit_duration", "fail", sploitID.Service, sploitID.Name),
		),
	}

	c.sploitMetrics[sploitID] = m

	return m
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

	m = &checkerMetrics{
		SuccessCheckCounter: metrics.NewCounter(genMetricName("checker_runs_total", "success", "check")),
		SuccessCheckPoints:  metrics.NewCounter(genMetricName("checker_points", "success", "check")),
		FailCheckCounter:    metrics.NewCounter(genMetricName("checker_runs_total", "fail", "check")),
		FailCheckPenalty:    metrics.NewCounter(genMetricName("checker_points", "fail", "check")),

		SuccessGetCounter: metrics.NewCounter(genMetricName("checker_runs_total", "success", "get")),
		SuccessGetPoints:  metrics.NewCounter(genMetricName("checker_points", "success", "get")),
		FailGetCounter:    metrics.NewCounter(genMetricName("checker_runs_total", "fail", "get")),
		FailGetPenalty:    metrics.NewCounter(genMetricName("checker_points", "fail", "get")),

		SuccessCheckDuration: metrics.NewHistogram(genMetricName("checker_run_duration", "success", "check")),
		FailCheckDuration:    metrics.NewHistogram(genMetricName("checker_run_duration", "fail", "check")),

		SuccessGetDuration: metrics.NewHistogram(genMetricName("checker_run_duration", "success", "get")),
		FailGetDuration:    metrics.NewHistogram(genMetricName("checker_run_duration", "fail", "get")),
	}

	c.checkerMetrics[checkerID] = m

	return m
}
