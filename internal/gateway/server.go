package gateway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Wh1stle05/AI-Gateway/internal/config"
	"github.com/Wh1stle05/AI-Gateway/internal/handler"
	"github.com/Wh1stle05/AI-Gateway/internal/metrics"
	"github.com/Wh1stle05/AI-Gateway/internal/middleware"
	"github.com/Wh1stle05/AI-Gateway/internal/ratelimit"
	"github.com/Wh1stle05/AI-Gateway/internal/router"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	cfg     *config.Config
	router  *router.Router
	limiter *ratelimit.Limiter
	redis   *redis.Client
	http    *http.Server
}

func New(cfg *config.Config) (*Server, error) {
	r, err := router.New(cfg)
	if err != nil {
		return nil, err
	}

	s := &Server{cfg: cfg, router: r}

	if cfg.RateLimit.Enabled {
		opts, err := redis.ParseURL(cfg.RateLimit.RedisURL)
		if err != nil {
			return nil, fmt.Errorf("rate_limit.redis_url: %w", err)
		}
		client := redis.NewClient(opts)
		if err := client.Ping(context.Background()).Err(); err != nil {
			return nil, fmt.Errorf("redis ping: %w", err)
		}
		s.redis = client
		s.limiter = ratelimit.New(client, ratelimit.Config{
			RequestsPerMinute: cfg.RateLimit.RequestsPerMinute,
			Burst:             cfg.RateLimit.Burst,
		})
	}

	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(metrics.Middleware)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)
	if s.limiter != nil {
		mux.Use(middleware.RateLimit(s.limiter))
	}

	health := handler.NewHealthHandler(r, s.limiter)
	chat := handler.NewChatHandler(cfg, r)

	mux.Get("/health", health.Live)
	mux.Get("/ready", health.Ready)
	mux.Handle("/metrics", promhttp.Handler())
	mux.Post("/v1/chat/completions", chat.ServeHTTP)

	s.http = &http.Server{
		Addr:    cfg.Server.Addr,
		Handler: mux,
	}
	return s, nil
}

func (s *Server) ListenAndServe() error {
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.redis != nil {
		_ = s.redis.Close()
	}
	return s.http.Shutdown(ctx)
}
