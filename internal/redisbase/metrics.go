package redisbase

import (
	"fmt"

	"github.com/VictoriaMetrics/metrics"
)

// storageMetrics holds all metrics for a storage instance.
type storageMetrics struct {
	SaveDuration             *metrics.Histogram
	GetByIDDuration          *metrics.Histogram
	GetRandomDuration        *metrics.Histogram
	GetRandomAttempts        *metrics.Histogram
	CleanupBatchSize         *metrics.Gauge
	CleanupReturnedBatchSize *metrics.Histogram
	CleanupDeleteDuration    *metrics.Histogram
}

// newStorageMetrics creates and initializes all metrics for a storage instance.
func newStorageMetrics(storageName string) *storageMetrics {
	genMetricName := func(name string) string {
		return fmt.Sprintf(`checker_redis_storage_%s{storage_name=%q}`, name, storageName)
	}

	return &storageMetrics{
		SaveDuration:             metrics.NewHistogram(genMetricName("save_duration")),
		GetByIDDuration:          metrics.NewHistogram(genMetricName("get_by_id_duration")),
		GetRandomDuration:        metrics.NewHistogram(genMetricName("get_random_duration")),
		GetRandomAttempts:        metrics.NewHistogram(genMetricName("get_random_attempts")),
		CleanupBatchSize:         metrics.NewGauge(genMetricName("cleanup_batch_size"), nil),
		CleanupReturnedBatchSize: metrics.NewHistogram(genMetricName("cleanup_returned_batch_size")),
		CleanupDeleteDuration:    metrics.NewHistogram(genMetricName("cleanup_delete_duration")),
	}
}

