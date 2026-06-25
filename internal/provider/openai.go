package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wh1stle05/AI-Gateway/internal/config"
	"github.com/Wh1stle05/AI-Gateway/internal/metrics"
	"github.com/Wh1stle05/AI-Gateway/internal/model"
)

type OpenAICompatible struct {
	name   string
	cfg    config.ProviderConfig
	client *http.Client
}

func NewOpenAICompatible(cfg config.ProviderConfig, client *http.Client) *OpenAICompatible {
	if client == nil {
		timeout, _ := time.ParseDuration(cfg.Timeout)
		if timeout == 0 {
			timeout = 120 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	return &OpenAICompatible{name: cfg.Name, cfg: cfg, client: client}
}

func (p *OpenAICompatible) Name() string {
	return p.name
}

func (p *OpenAICompatible) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.cfg.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(httpReq)

	start := time.Now()
	resp, err := p.client.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		metrics.RecordProviderRequest(p.name, req.Model, "error", duration)
		return nil, fmt.Errorf("upstream request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		metrics.RecordProviderRequest(p.name, req.Model, "error", duration)
		return nil, fmt.Errorf("read upstream response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		metrics.RecordProviderRequest(p.name, req.Model, strconv.Itoa(resp.StatusCode), duration)
		return nil, parseUpstreamError(resp.StatusCode, respBody)
	}

	var out model.ChatCompletionResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		metrics.RecordProviderRequest(p.name, req.Model, "error", duration)
		return nil, fmt.Errorf("decode upstream response: %w", err)
	}
	metrics.RecordProviderRequest(p.name, req.Model, "200", duration)
	return &out, nil
}

func (p *OpenAICompatible) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (*http.Response, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.cfg.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	p.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	start := time.Now()
	resp, err := p.client.Do(httpReq)
	duration := time.Since(start)
	if err != nil {
		metrics.RecordProviderRequest(p.name, req.Model, "error", duration)
		return nil, fmt.Errorf("upstream request: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		respBody, readErr := io.ReadAll(resp.Body)
		metrics.RecordProviderRequest(p.name, req.Model, strconv.Itoa(resp.StatusCode), duration)
		if readErr != nil {
			return nil, fmt.Errorf("upstream status %d: read body: %w", resp.StatusCode, readErr)
		}
		return nil, parseUpstreamError(resp.StatusCode, respBody)
	}
	metrics.RecordProviderRequest(p.name, req.Model, "200", duration)
	return resp, nil
}

func (p *OpenAICompatible) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if p.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	}
}

type UpstreamError struct {
	Status int
	Body   []byte
}

func (e *UpstreamError) Error() string {
	msg := strings.TrimSpace(string(e.Body))
	if msg == "" {
		return fmt.Sprintf("upstream error: status %d", e.Status)
	}
	return fmt.Sprintf("upstream error: status %d: %s", e.Status, msg)
}

func parseUpstreamError(status int, body []byte) error {
	return &UpstreamError{Status: status, Body: body}
}

func StatusCode(err error) (int, bool) {
	var ue *UpstreamError
	if errors.As(err, &ue) {
		return ue.Status, true
	}
	return 0, false
}

func UpstreamBody(err error) []byte {
	var ue *UpstreamError
	if errors.As(err, &ue) {
		return ue.Body
	}
	return nil
}

// IsFailure reports whether an upstream error should trip the circuit breaker.
// Client errors (4xx) are not counted as provider failures.
func IsFailure(err error) bool {
	if err == nil {
		return false
	}
	var ue *UpstreamError
	if errors.As(err, &ue) {
		return ue.Status >= http.StatusInternalServerError
	}
	return true
}
