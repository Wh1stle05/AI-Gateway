package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Wh1stle05/AI-Gateway/internal/ratelimit"
	"github.com/Wh1stle05/AI-Gateway/internal/router"
)

type HealthHandler struct {
	router  *router.Router
	limiter *ratelimit.Limiter
}

func NewHealthHandler(r *router.Router, limiter *ratelimit.Limiter) *HealthHandler {
	return &HealthHandler{router: r, limiter: limiter}
}

func (h *HealthHandler) Live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if h.router.ProviderCount() == 0 {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
			"reason": "no providers configured",
		})
		return
	}

	if h.limiter != nil {
		if err := h.limiter.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status": "not_ready",
				"reason": "redis unavailable",
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("encode json", "error", err)
	}
}
