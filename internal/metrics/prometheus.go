package metrics

import (
	"sync"

	"github.com/josepht96/scout/internal/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusExporter exports Scout metrics to Prometheus
type PrometheusExporter struct {
	testStatus             *prometheus.GaugeVec
	testLatency            *prometheus.GaugeVec
	collectionLastRun      *prometheus.GaugeVec
	collectionLastSuccess  *prometheus.GaugeVec
	collectionDuration     *prometheus.GaugeVec
	collectionTestTotal    *prometheus.GaugeVec
	mu                     sync.RWMutex
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter() *PrometheusExporter {
	return &PrometheusExporter{
		testStatus: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "scout_test_status",
				Help: "Test status (1 for pass, 0 for fail)",
			},
			[]string{"collection", "test_name", "url", "method"},
		),
		testLatency: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "scout_test_latency_ms",
				Help: "Test response time in milliseconds",
			},
			[]string{"collection", "test_name", "url", "method"},
		),
		collectionLastRun: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "scout_collection_last_run_timestamp",
				Help: "Timestamp of the last run for each collection",
			},
			[]string{"collection"},
		),
		collectionLastSuccess: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "scout_collection_last_success_timestamp",
				Help: "Timestamp of the last successful run (all tests passed) for each collection",
			},
			[]string{"collection"},
		),
		collectionDuration: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "scout_collection_duration_ms",
				Help: "Duration of collection execution in milliseconds",
			},
			[]string{"collection"},
		),
		collectionTestTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "scout_collection_tests_total",
				Help: "Total number of tests in collection",
			},
			[]string{"collection", "status"},
		),
	}
}

// UpdateMetrics updates Prometheus metrics with the latest results
func (e *PrometheusExporter) UpdateMetrics(results *storage.LatestResults) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Reset all metrics before updating
	e.testStatus.Reset()
	e.testLatency.Reset()
	e.collectionLastRun.Reset()
	e.collectionLastSuccess.Reset()
	e.collectionDuration.Reset()
	e.collectionTestTotal.Reset()

	// Update metrics for each collection across all groups
	for _, group := range results.EnvironmentGroups {
		for _, cr := range group.Collections {
			collectionName := cr.Collection.Name

			// If there's no execution yet, skip
			if cr.Execution == nil {
				continue
			}

			// Update collection-level metrics
			e.collectionLastRun.WithLabelValues(collectionName).Set(
				float64(cr.Execution.StartedAt.Unix()),
			)

			// Update last success timestamp only if all tests passed
			if cr.Execution.FailedTests == 0 && cr.Execution.TotalTests > 0 {
				e.collectionLastSuccess.WithLabelValues(collectionName).Set(
					float64(cr.Execution.StartedAt.Unix()),
				)
			}

			e.collectionDuration.WithLabelValues(collectionName).Set(
				float64(cr.Execution.DurationMs),
			)

			e.collectionTestTotal.WithLabelValues(collectionName, "total").Set(
				float64(cr.Execution.TotalTests),
			)

			e.collectionTestTotal.WithLabelValues(collectionName, "passed").Set(
				float64(cr.Execution.PassedTests),
			)

			e.collectionTestTotal.WithLabelValues(collectionName, "failed").Set(
				float64(cr.Execution.FailedTests),
			)

			// Update test-level metrics
			for _, result := range cr.Results {
			// Get labels
			testName := result.TestName
			url := ""
			method := ""

			if result.URL != nil {
				url = *result.URL
			}
			if result.Method != nil {
				method = *result.Method
			}

			// Update test status
			statusValue := 0.0
			if result.Passed {
				statusValue = 1.0
			}
			e.testStatus.WithLabelValues(collectionName, testName, url, method).Set(statusValue)

				// Update test latency if available
				if result.ResponseTimeMs != nil {
					e.testLatency.WithLabelValues(collectionName, testName, url, method).Set(
						float64(*result.ResponseTimeMs),
					)
				}
			}
		}
	}
}

// GetRegistry returns the Prometheus registry (for custom metrics)
func (e *PrometheusExporter) GetRegistry() *prometheus.Registry {
	return prometheus.DefaultRegisterer.(*prometheus.Registry)
}
