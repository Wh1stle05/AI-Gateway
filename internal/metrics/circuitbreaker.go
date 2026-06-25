package metrics

func SetCircuitBreakerState(provider, state string) {
	var val float64
	switch state {
	case "closed":
		val = 0
	case "open":
		val = 1
	case "half_open":
		val = 2
	}
	CircuitBreakerState.WithLabelValues(provider).Set(val)
}

func RecordCircuitBreakerTrip(provider string) {
	CircuitBreakerTripsTotal.WithLabelValues(provider).Inc()
}

func RecordCircuitBreakerRejection(provider string) {
	CircuitBreakerRejectionsTotal.WithLabelValues(provider).Inc()
}
