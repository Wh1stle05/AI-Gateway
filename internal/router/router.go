package router

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Wh1stle05/AI-Gateway/internal/circuitbreaker"
	"github.com/Wh1stle05/AI-Gateway/internal/config"
	"github.com/Wh1stle05/AI-Gateway/internal/metrics"
	"github.com/Wh1stle05/AI-Gateway/internal/model"
	"github.com/Wh1stle05/AI-Gateway/internal/provider"
)

type Router struct {
	cfg       *config.Config
	providers map[string]provider.Provider
	breakers  map[string]*circuitbreaker.Breaker
}

func New(cfg *config.Config) (*Router, error) {
	providers := make(map[string]provider.Provider, len(cfg.Providers))
	breakers := make(map[string]*circuitbreaker.Breaker)
	streamClient := &http.Client{}

	var cbCfg circuitbreaker.Config
	if cfg.CircuitBreaker.Enabled {
		threshold, openTimeout, halfOpenMax, err := cfg.CircuitBreakerSettings()
		if err != nil {
			return nil, err
		}
		cbCfg = circuitbreaker.Config{
			FailureThreshold:    threshold,
			OpenTimeout:         openTimeout,
			HalfOpenMaxRequests: halfOpenMax,
		}
	}

	for _, pcfg := range cfg.Providers {
		if _, exists := providers[pcfg.Name]; exists {
			continue
		}
		p, err := provider.New(pcfg, streamClient)
		if err != nil {
			return nil, fmt.Errorf("provider %q: %w", pcfg.Name, err)
		}
		if cfg.CircuitBreaker.Enabled {
			br, ok := breakers[pcfg.Name]
			if !ok {
				br = circuitbreaker.New(cbCfg)
				breakers[pcfg.Name] = br
			}
			p = circuitbreaker.Wrap(p, br)
		}
		providers[pcfg.Name] = p
	}

	return &Router{cfg: cfg, providers: providers, breakers: breakers}, nil
}

type RouteResult struct {
	Provider provider.Provider
	Fallback provider.Provider
}

func (r *Router) Resolve(model string) (*RouteResult, error) {
	pcfg, fallbackName, err := r.cfg.ProviderForModel(model)
	if err != nil {
		return nil, err
	}

	primary, ok := r.providers[pcfg.Name]
	if !ok {
		return nil, fmt.Errorf("provider %q not initialized", pcfg.Name)
	}

	var fallback provider.Provider
	if fallbackName != "" {
		fallback, ok = r.providers[fallbackName]
		if !ok {
			return nil, fmt.Errorf("fallback provider %q not initialized", fallbackName)
		}
	}

	return &RouteResult{Provider: primary, Fallback: fallback}, nil
}

func (r *Router) ChatCompletion(ctx context.Context, req *model.ChatCompletionRequest) (*model.ChatCompletionResponse, error) {
	route, err := r.Resolve(req.Model)
	if err != nil {
		return nil, err
	}

	resp, err := route.Provider.ChatCompletion(ctx, req)
	if err == nil || route.Fallback == nil {
		return resp, err
	}

	fallbackResp, fallbackErr := route.Fallback.ChatCompletion(ctx, req)
	if fallbackErr != nil {
		return nil, fmt.Errorf("primary failed (%v); fallback failed (%v)", err, fallbackErr)
	}
	metrics.RecordFallback(req.Model, route.Provider.Name(), route.Fallback.Name())
	return fallbackResp, nil
}

func (r *Router) ChatCompletionStream(ctx context.Context, req *model.ChatCompletionRequest) (*http.Response, provider.Provider, error) {
	route, err := r.Resolve(req.Model)
	if err != nil {
		return nil, nil, err
	}

	resp, err := route.Provider.ChatCompletionStream(ctx, req)
	if err == nil {
		return resp, route.Provider, nil
	}
	if route.Fallback == nil {
		return nil, nil, err
	}

	fallbackResp, fallbackErr := route.Fallback.ChatCompletionStream(ctx, req)
	if fallbackErr != nil {
		return nil, nil, fmt.Errorf("primary failed (%v); fallback failed (%v)", err, fallbackErr)
	}
	metrics.RecordFallback(req.Model, route.Provider.Name(), route.Fallback.Name())
	return fallbackResp, route.Fallback, nil
}

func (r *Router) ProviderCount() int {
	return len(r.providers)
}

func (r *Router) BreakerState(name string) (string, bool) {
	br, ok := r.breakers[name]
	if !ok {
		return "", false
	}
	return br.State(), true
}
