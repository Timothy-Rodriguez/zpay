package pkg

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zpay_http_requests_total",
			Help: "Total number of HTTP requests handled by the backend.",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "zpay_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	TransactionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zpay_transactions_total",
			Help: "Total number of transaction attempts by result.",
		},
		[]string{"result"},
	)

	HTTPInFlightRequests = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "zpay_http_in_flight_requests",
			Help: "Current number of in-flight HTTP requests.",
		},
	)
)
