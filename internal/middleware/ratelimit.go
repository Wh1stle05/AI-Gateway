package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wh1stle05/AI-Gateway/internal/model"
	"github.com/Wh1stle05/AI-Gateway/internal/ratelimit"
)

func RateLimit(limiter *ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := rateLimitKey(r)
			allowed, err := limiter.Allow(r.Context(), key)
			if err != nil {
				writeRateLimitError(w, http.StatusServiceUnavailable, "rate_limit_unavailable", "Rate limiter unavailable")
				return
			}
			if !allowed {
				w.Header().Set("Retry-After", strconv.Itoa(limiter.RetryAfterSeconds()))
				writeRateLimitError(w, http.StatusTooManyRequests, "rate_limit_exceeded", "Rate limit exceeded")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitKey(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return "auth:" + strings.TrimPrefix(auth, "Bearer ")
	}
	if key := r.Header.Get("X-API-Key"); key != "" {
		return "key:" + key
	}
	host, _, _ := strings.Cut(r.RemoteAddr, ":")
	if host == "" {
		return "ip:unknown"
	}
	return "ip:" + host
}

func writeRateLimitError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.ErrorResponse{
		Error: model.ErrorDetail{
			Message: message,
			Type:    errType,
		},
	})
}
