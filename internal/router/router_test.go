package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/Wh1stle05/AI-Gateway/internal/config"
	"github.com/Wh1stle05/AI-Gateway/internal/model"
)

func TestChatCompletionFallback(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"primary down"}`))
	}))
	defer primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(model.ChatCompletionResponse{
			ID:      "chatcmpl-test",
			Object:  "chat.completion",
			Created: 1,
			Model:   "demo",
			Choices: []model.Choice{{
				Index: 0,
				Message: model.Message{
					Role:    "assistant",
					Content: "fallback ok",
				},
			}},
		})
	}))
	defer fallback.Close()

	cfg := &config.Config{
		Providers: []config.ProviderConfig{
			{Name: "primary", BaseURL: primary.URL + "/v1", Models: []string{"demo"}},
			{Name: "fallback", BaseURL: fallback.URL + "/v1", Models: []string{"demo-fb"}},
		},
		Routing: []config.RouteConfig{{
			Model:    "demo",
			Provider: "primary",
			Fallback: "fallback",
		}},
	}

	r, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := r.ChatCompletion(context.Background(), &model.ChatCompletionRequest{
		Model:    "demo",
		Messages: []model.Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("ChatCompletion() error = %v", err)
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content != "fallback ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCircuitBreakerTripsToFallback(t *testing.T) {
	var primaryHits atomic.Int32
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"primary down"}`))
	}))
	defer primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(model.ChatCompletionResponse{
			ID:     "chatcmpl-fallback",
			Object: "chat.completion",
			Model:  "demo",
			Choices: []model.Choice{{
				Message: model.Message{Role: "assistant", Content: "fallback ok"},
			}},
		})
	}))
	defer fallback.Close()

	cfg := &config.Config{
		CircuitBreaker: config.CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 3,
			OpenTimeout:      "1h",
		},
		Providers: []config.ProviderConfig{
			{Name: "primary", BaseURL: primary.URL + "/v1", Models: []string{"demo"}},
			{Name: "fallback", BaseURL: fallback.URL + "/v1", Models: []string{"demo-fb"}},
		},
		Routing: []config.RouteConfig{{
			Model:    "demo",
			Provider: "primary",
			Fallback: "fallback",
		}},
	}

	r, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	req := &model.ChatCompletionRequest{
		Model:    "demo",
		Messages: []model.Message{{Role: "user", Content: "hi"}},
	}

	for i := 0; i < 3; i++ {
		if _, err := r.ChatCompletion(context.Background(), req); err != nil {
			t.Fatalf("warmup request %d failed: %v", i, err)
		}
	}

	state, ok := r.BreakerState("primary")
	if !ok || state != "open" {
		t.Fatalf("breaker state = %q, want open", state)
	}

	hitsBefore := primaryHits.Load()
	resp, err := r.ChatCompletion(context.Background(), req)
	if err != nil {
		t.Fatalf("ChatCompletion() after open breaker: %v", err)
	}
	if resp.Choices[0].Message.Content != "fallback ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if primaryHits.Load() != hitsBefore {
		t.Fatalf("primary should not be called when breaker is open")
	}
}
