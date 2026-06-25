package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aigateway_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aigateway_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "aigateway_http_requests_in_flight",
			Help: "Number of HTTP requests currently being processed",
		},
	)

	ChatRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aigateway_chat_requests_total",
			Help: "Total number of chat completion requests",
		},
		[]string{"model", "stream"},
	)

	ChatTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aigateway_chat_tokens_total",
			Help: "Total tokens used in chat completions",
		},
		[]string{"model", "type"},
	)

	ChatErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aigateway_chat_errors_total",
			Help: "Total number of chat completion errors",
		},
		[]string{"model", "error_type"},
	)

	ProviderRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aigateway_provider_requests_total",
			Help: "Total number of upstream provider requests",
		},
		[]string{"provider", "model", "status"},
	)

	ProviderRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aigateway_provider_request_duration_seconds",
			Help:    "Upstream provider request duration in seconds",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120},
		},
		[]string{"provider", "model"},
	)

	RouterFallbackTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aigateway_router_fallback_total",
			Help: "Total number of fallback route activations",
		},
		[]string{"model", "primary_provider", "fallback_provider"},
	)

	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "aigateway_circuit_breaker_state",
			Help: "Circuit breaker state (0=closed, 1=open, 2=half_open)",
		},
		[]string{"provider"},
	)

	CircuitBreakerTripsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aigateway_circuit_breaker_trips_total",
			Help: "Total number of circuit breaker trips",
		},
		[]string{"provider"},
	)

	CircuitBreakerRejectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aigateway_circuit_breaker_rejections_total",
			Help: "Total number of requests rejected by circuit breaker",
		},
		[]string{"provider"},
	)

	RateLimitAllowedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aigateway_ratelimit_allowed_total",
			Help: "Total number of requests allowed by rate limiter",
		},
		[]string{"key_type"},
	)

	RateLimitRejectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aigateway_ratelimit_rejected_total",
			Help: "Total number of requests rejected by rate limiter",
		},
		[]string{"key_type"},
	)
)
