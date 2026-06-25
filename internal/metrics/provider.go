package metrics

import (
	"time"
)

func RecordProviderRequest(provider, model, status string, duration time.Duration) {
	ProviderRequestsTotal.WithLabelValues(provider, model, status).Inc()
	ProviderRequestDuration.WithLabelValues(provider, model).Observe(duration.Seconds())
}

func RecordChatRequest(model string, stream bool) {
	ChatRequestsTotal.WithLabelValues(model, strconvBool(stream)).Inc()
}

func RecordChatTokens(model string, promptTokens, completionTokens int) {
	if promptTokens > 0 {
		ChatTokensTotal.WithLabelValues(model, "prompt").Add(float64(promptTokens))
	}
	if completionTokens > 0 {
		ChatTokensTotal.WithLabelValues(model, "completion").Add(float64(completionTokens))
	}
}

func RecordChatError(model, errorType string) {
	ChatErrorsTotal.WithLabelValues(model, errorType).Inc()
}

func RecordFallback(model, primary, fallback string) {
	RouterFallbackTotal.WithLabelValues(model, primary, fallback).Inc()
}

func strconvBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
