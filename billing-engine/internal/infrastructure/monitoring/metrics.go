package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type HTTPMetrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
}

type DBMetrics struct {
	QueryDuration *prometheus.HistogramVec
}

type BusinessMetrics struct {
	LoansCreatedTotal      prometheus.Counter
	PaymentsProcessedTotal *prometheus.CounterVec
}

var (
	HTTP = HTTPMetrics{
		RequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "billing_engine_http_requests_total",
				Help: "Total number of HTTP requests received.",
			},
			[]string{"method", "path", "code"},
		),
		RequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "billing_engine_http_request_duration_seconds",
				Help:    "Histogram of HTTP request latencies.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "code"},
		),
	}

	DB = DBMetrics{
		QueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "billing_engine_db_query_duration_seconds",
				Help:    "Histogram of database query latencies.",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"query_name", "status"},
		),
	}

	Business = BusinessMetrics{
		LoansCreatedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "billing_engine_loans_created_total",
				Help: "Total number of loans successfully created.",
			},
		),
		PaymentsProcessedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "billing_engine_payments_processed_total",
				Help: "Total number of payment processing attempts.",
			},
			[]string{"status"},
		),
	}
)

func RecordHTTPRequest(method, path, code string, duration time.Duration) {
	HTTP.RequestsTotal.WithLabelValues(method, path, code).Inc()
	HTTP.RequestDuration.WithLabelValues(method, path, code).Observe(duration.Seconds())
}

func RecordDBQuery(queryName, status string, duration time.Duration) {
	DB.QueryDuration.WithLabelValues(queryName, status).Observe(duration.Seconds())
}

func RecordLoanCreation() {
	Business.LoansCreatedTotal.Inc()
}

func RecordPayment(status string) {
	Business.PaymentsProcessedTotal.WithLabelValues(status).Inc()
}
