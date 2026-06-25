package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Wh1stle05/AI-Gateway/internal/config"
	"github.com/Wh1stle05/AI-Gateway/internal/middleware"
	"github.com/Wh1stle05/AI-Gateway/internal/model"
	"github.com/Wh1stle05/AI-Gateway/internal/provider"
	"github.com/Wh1stle05/AI-Gateway/internal/router"
)

type ChatHandler struct {
	cfg    *config.Config
	router *router.Router
}

func NewChatHandler(cfg *config.Config, r *router.Router) *ChatHandler {
	return &ChatHandler{cfg: cfg, router: r}
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.cfg.GatewayAuthEnabled() {
		auth := r.Header.Get("Authorization")
		want := "Bearer " + h.cfg.Gateway.APIKey
		if auth != want {
			writeError(w, http.StatusUnauthorized, "invalid_api_key", "Invalid gateway API key")
			return
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Failed to read request body")
		return
	}

	var req model.ChatCompletionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid JSON body")
		return
	}
	if req.Model == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "model is required")
		return
	}
	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "messages are required")
		return
	}

	if req.Stream {
		h.handleStream(w, r, &req)
		return
	}
	h.handleJSON(w, r, &req)
}

func (h *ChatHandler) handleJSON(w http.ResponseWriter, r *http.Request, req *model.ChatCompletionRequest) {
	resp, err := h.router.ChatCompletion(r.Context(), req)
	if err != nil {
		h.writeRouteError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", middleware.GetRequestID(r.Context()))
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("encode response", "error", err, "request_id", middleware.GetRequestID(r.Context()))
	}
}

func (h *ChatHandler) handleStream(w http.ResponseWriter, r *http.Request, req *model.ChatCompletionRequest) {
	upstream, usedProvider, err := h.router.ChatCompletionStream(r.Context(), req)
	if err != nil {
		h.writeRouteError(w, r, err)
		return
	}
	defer upstream.Body.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal_error", "Streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Request-ID", middleware.GetRequestID(r.Context()))
	w.Header().Set("X-Provider", usedProvider.Name())
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	if _, err := io.Copy(w, upstream.Body); err != nil {
		slog.Error("stream copy", "error", err, "request_id", middleware.GetRequestID(r.Context()))
	}
}

func (h *ChatHandler) writeRouteError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.GetRequestID(r.Context())
	slog.Error("chat completion failed", "error", err, "request_id", requestID)

	if status, ok := provider.StatusCode(err); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-ID", requestID)
		w.WriteHeader(status)
		if body := provider.UpstreamBody(err); len(body) > 0 {
			_, _ = w.Write(body)
			return
		}
		writeError(w, status, "upstream_error", err.Error())
		return
	}

	msg := err.Error()
	if strings.Contains(msg, "no provider configured") || strings.Contains(msg, "requires") {
		writeError(w, http.StatusBadRequest, "invalid_request", msg)
		return
	}

	writeError(w, http.StatusBadGateway, "upstream_error", msg)
}

func writeError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(model.ErrorResponse{
		Error: model.ErrorDetail{
			Message: message,
			Type:    errType,
		},
	})
}
