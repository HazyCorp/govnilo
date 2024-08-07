package checkerctrl

import (
	"fmt"

	"github.com/VictoriaMetrics/metrics"

	"github.com/HazyCorp/govnilo/pkg/hazycheck"
)

type checkerMetrics struct {
	SuccessCheckCounter *metrics.Counter
	FailCheckCounter    *metrics.Counter

	SuccessGetCounter *metrics.Counter
	FailGetCounter    *metrics.Counter

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

func (c *Controller) sploitMetricsFor(sploitID hazycheck.SploitID) *sploitMetrics {
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

func (c *Controller) checkerMetricsFor(checkerID hazycheck.CheckerID) *checkerMetrics {
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
	m = &checkerMetrics{
		SuccessCheckCounter: metrics.NewCounter(
			fmt.Sprintf(
				template,
				"checker_runs_total",
				"success",
				"check",
				checkerID.Service,
				checkerID.Name,
			),
		),
		FailCheckCounter: metrics.NewCounter(
			fmt.Sprintf(
				template,
				"checker_runs_total",
				"fail",
				"check",
				checkerID.Service,
				checkerID.Name,
			),
		),
		SuccessGetCounter: metrics.NewCounter(
			fmt.Sprintf(
				template,
				"checker_runs_total",
				"success",
				"get",
				checkerID.Service,
				checkerID.Name,
			),
		),
		FailGetCounter: metrics.NewCounter(
			fmt.Sprintf(
				template,
				"checker_runs_total",
				"fail",
				"get",
				checkerID.Service,
				checkerID.Name,
			),
		),
		SuccessCheckDuration: metrics.NewHistogram(
			fmt.Sprintf(
				template,
				"checker_run_duration",
				"success",
				"check",
				checkerID.Service,
				checkerID.Name,
			),
		),
		FailCheckDuration: metrics.NewHistogram(
			fmt.Sprintf(
				template,
				"checker_run_duration",
				"fail",
				"check",
				checkerID.Service,
				checkerID.Name,
			),
		),
		SuccessGetDuration: metrics.NewHistogram(
			fmt.Sprintf(
				template,
				"checker_run_duration",
				"success",
				"get",
				checkerID.Service,
				checkerID.Name,
			),
		),
		FailGetDuration: metrics.NewHistogram(
			fmt.Sprintf(
				template,
				"checker_run_duration",
				"fail",
				"get",
				checkerID.Service,
				checkerID.Name,
			),
		),
	}

	c.checkerMetrics[checkerID] = m

	return m
}
