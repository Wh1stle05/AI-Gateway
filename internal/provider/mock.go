package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wh1stle05/AI-Gateway/internal/config"
	"github.com/Wh1stle05/AI-Gateway/internal/model"
)

type Mock struct {
	name string
}

func NewMock(cfg config.ProviderConfig) *Mock {
	return &Mock{name: cfg.Name}
}

func (p *Mock) Name() string {
	return p.name
}

func (p *Mock) ChatCompletion(_ context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	content := "mock response for " + req.Model
	return &model.ChatCompletionResponse{
		ID:      "chatcmpl-mock",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []model.Choice{{
			Index: 0,
			Message: model.Message{
				Role:    "assistant",
				Content: content,
			},
			FinishReason: strPtr("stop"),
		}},
		Usage: &model.Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}, nil
}

func (p *Mock) ChatCompletionStream(_ context.Context, req *model.ChatCompletionRequest) (*http.Response, error) {
	content := "mock stream for " + req.Model
	chunk := model.ChatCompletionResponse{
		ID:      "chatcmpl-mock",
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []model.Choice{{
			Index: 0,
			Delta: model.Message{Role: "assistant", Content: content},
		}},
	}
	payload, err := json.Marshal(chunk)
	if err != nil {
		return nil, err
	}

	body := strings.Join([]string{
		"data: " + string(payload),
		"",
		"data: [DONE]",
		"",
	}, "\n")

	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       ioNopCloser{Reader: strings.NewReader(body)},
	}, nil
}

type ioNopCloser struct {
	io.Reader
}

func (ioNopCloser) Close() error { return nil }

func strPtr(v string) *string { return &v }

var _ Provider = (*Mock)(nil)
