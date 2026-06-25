package circuitbreaker

import (
	"context"
	"net/http"

	"github.com/Wh1stle05/AI-Gateway/internal/model"
	"github.com/Wh1stle05/AI-Gateway/internal/provider"
)

type Provider struct {
	inner   provider.Provider
	breaker *Breaker
}

func Wrap(inner provider.Provider, breaker *Breaker) *Provider {
	return &Provider{inner: inner, breaker: breaker}
}

func (p *Provider) Name() string {
	return p.inner.Name()
}

func (p *Provider) BreakerState() string {
	return p.breaker.State()
}

func (p *Provider) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	if !p.breaker.Allow() {
		return nil, ErrOpen
	}

	resp, err := p.inner.ChatCompletion(ctx, req)
	p.record(err)
	return resp, err
}

func (p *Provider) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (*http.Response, error) {
	if !p.breaker.Allow() {
		return nil, ErrOpen
	}

	resp, err := p.inner.ChatCompletionStream(ctx, req)
	if err != nil {
		p.record(err)
		return nil, err
	}
	return resp, err
}

func (p *Provider) record(err error) {
	if provider.IsFailure(err) {
		p.breaker.RecordFailure()
		return
	}
	p.breaker.RecordSuccess()
}
