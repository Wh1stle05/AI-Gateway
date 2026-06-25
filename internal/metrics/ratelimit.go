package metrics

func RecordRateLimitAllowed(keyType string) {
	RateLimitAllowedTotal.WithLabelValues(keyType).Inc()
}

func RecordRateLimitRejected(keyType string) {
	RateLimitRejectedTotal.WithLabelValues(keyType).Inc()
}
