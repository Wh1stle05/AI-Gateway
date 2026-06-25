package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server         ServerConfig         `yaml:"server"`
	Gateway        GatewayConfig        `yaml:"gateway"`
	Metrics        MetricsConfig        `yaml:"metrics"`
	RateLimit      RateLimitConfig      `yaml:"rate_limit"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
	Providers      []ProviderConfig     `yaml:"providers"`
	Routing        []RouteConfig        `yaml:"routing"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type GatewayConfig struct {
	APIKey string `yaml:"api_key"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type RateLimitConfig struct {
	Enabled           bool   `yaml:"enabled"`
	RedisURL          string `yaml:"redis_url"`
	RequestsPerMinute int    `yaml:"requests_per_minute"`
	Burst             int    `yaml:"burst"`
}

type CircuitBreakerConfig struct {
	Enabled             bool   `yaml:"enabled"`
	FailureThreshold    int    `yaml:"failure_threshold"`
	OpenTimeout         string `yaml:"open_timeout"`
	HalfOpenMaxRequests int    `yaml:"half_open_max_requests"`
}

type ProviderConfig struct {
	Name    string   `yaml:"name"`
	Type    string   `yaml:"type"`
	BaseURL string   `yaml:"base_url"`
	APIKey  string   `yaml:"api_key"`
	Models  []string `yaml:"models"`
	Timeout string   `yaml:"timeout"`
}

type RouteConfig struct {
	Model    string `yaml:"model"`
	Provider string `yaml:"provider"`
	Fallback string `yaml:"fallback,omitempty"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.expandEnv(); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Addr == "" {
		c.Server.Addr = ":8080"
	}
	if c.Metrics.Path == "" {
		c.Metrics.Path = "/metrics"
	}
	if c.RateLimit.RequestsPerMinute == 0 {
		c.RateLimit.RequestsPerMinute = 60
	}
	if c.RateLimit.Burst == 0 {
		c.RateLimit.Burst = 10
	}
	if c.RateLimit.RedisURL == "" {
		c.RateLimit.RedisURL = "redis://127.0.0.1:6379/0"
	}
	if c.CircuitBreaker.FailureThreshold == 0 {
		c.CircuitBreaker.FailureThreshold = 5
	}
	if c.CircuitBreaker.OpenTimeout == "" {
		c.CircuitBreaker.OpenTimeout = "30s"
	}
	if c.CircuitBreaker.HalfOpenMaxRequests == 0 {
		c.CircuitBreaker.HalfOpenMaxRequests = 2
	}
	for i := range c.Providers {
		if c.Providers[i].Timeout == "" {
			c.Providers[i].Timeout = "120s"
		}
		if c.Providers[i].Type == "" {
			c.Providers[i].Type = "openai"
		}
	}
}

func (c *Config) expandEnv() error {
	c.Gateway.APIKey = expandString(c.Gateway.APIKey)
	c.RateLimit.RedisURL = expandString(c.RateLimit.RedisURL)
	for i := range c.Providers {
		c.Providers[i].APIKey = expandString(c.Providers[i].APIKey)
		c.Providers[i].BaseURL = expandString(c.Providers[i].BaseURL)
	}
	return nil
}

func expandString(s string) string {
	return os.Expand(s, func(key string) string {
		return os.Getenv(key)
	})
}

func (c *Config) validate() error {
	if len(c.Providers) == 0 {
		return fmt.Errorf("at least one provider is required")
	}

	names := make(map[string]struct{}, len(c.Providers))
	for _, p := range c.Providers {
		if p.Name == "" {
			return fmt.Errorf("provider name is required")
		}
		if p.Type != "mock" && p.BaseURL == "" {
			return fmt.Errorf("provider %q: base_url is required", p.Name)
		}
		if _, ok := names[p.Name]; ok {
			return fmt.Errorf("duplicate provider name %q", p.Name)
		}
		names[p.Name] = struct{}{}
	}

	for _, r := range c.Routing {
		if r.Model == "" {
			return fmt.Errorf("routing entry requires model")
		}
		if r.Provider == "" {
			return fmt.Errorf("routing for model %q requires provider", r.Model)
		}
		if _, ok := names[r.Provider]; !ok {
			return fmt.Errorf("routing for model %q: unknown provider %q", r.Model, r.Provider)
		}
		if r.Fallback != "" {
			if _, ok := names[r.Fallback]; !ok {
				return fmt.Errorf("routing for model %q: unknown fallback %q", r.Model, r.Fallback)
			}
		}
	}

	return nil
}

func (c *Config) ProviderByName(name string) (*ProviderConfig, bool) {
	for i := range c.Providers {
		if c.Providers[i].Name == name {
			return &c.Providers[i], true
		}
	}
	return nil, false
}

func (c *Config) RouteForModel(model string) (*RouteConfig, bool) {
	for i := range c.Routing {
		if c.Routing[i].Model == model {
			return &c.Routing[i], true
		}
	}
	return nil, false
}

func (c *Config) ProviderForModel(model string) (*ProviderConfig, string, error) {
	if route, ok := c.RouteForModel(model); ok {
		p, found := c.ProviderByName(route.Provider)
		if !found {
			return nil, "", fmt.Errorf("provider %q not found", route.Provider)
		}
		return p, route.Fallback, nil
	}

	for i := range c.Providers {
		for _, m := range c.Providers[i].Models {
			if m == model {
				return &c.Providers[i], "", nil
			}
		}
	}

	return nil, "", fmt.Errorf("no provider configured for model %q", model)
}

func (c *Config) GatewayAuthEnabled() bool {
	return strings.TrimSpace(c.Gateway.APIKey) != ""
}

func (c *Config) CircuitBreakerSettings() (failureThreshold int, openTimeout time.Duration, halfOpenMax int, err error) {
	openTimeout, err = time.ParseDuration(c.CircuitBreaker.OpenTimeout)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("circuit_breaker.open_timeout: %w", err)
	}
	return c.CircuitBreaker.FailureThreshold, openTimeout, c.CircuitBreaker.HalfOpenMaxRequests, nil
}
