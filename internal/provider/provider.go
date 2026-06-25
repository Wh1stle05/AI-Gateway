package provider

import (
	"context"
	"net/http"

	"github.com/Wh1stle05/AI-Gateway/internal/config"
	"github.com/Wh1stle05/AI-Gateway/internal/model"
)

type Provider interface {
	Name() string
	ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error)
	ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (*http.Response, error)
}

func New(cfg config.ProviderConfig, client *http.Client) (Provider, error) {
	switch cfg.Type {
	case "mock":
		return NewMock(cfg), nil
	case "", "openai", "ollama":
		return NewOpenAICompatible(cfg, client), nil
	default:
		return nil, &UnsupportedTypeError{Type: cfg.Type}
	}
}

type UnsupportedTypeError struct {
	Type string
}

func (e *UnsupportedTypeError) Error() string {
	return "unsupported provider type: " + e.Type
}
