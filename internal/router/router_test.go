package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
