package gateway

import (
	"context"
	"net/http"

	"github.com/Wh1stle05/AI-Gateway/internal/config"
	"github.com/Wh1stle05/AI-Gateway/internal/handler"
	"github.com/Wh1stle05/AI-Gateway/internal/middleware"
	"github.com/Wh1stle05/AI-Gateway/internal/router"
	"github.com/go-chi/chi/v5"
)

type Server struct {
	cfg    *config.Config
	router *router.Router
	http   *http.Server
}

func New(cfg *config.Config) (*Server, error) {
	r, err := router.New(cfg)
	if err != nil {
		return nil, err
	}

	s := &Server{cfg: cfg, router: r}
	mux := chi.NewRouter()
	mux.Use(middleware.RequestID)
	mux.Use(middleware.Logger)
	mux.Use(middleware.Recoverer)

	health := handler.NewHealthHandler(r)
	chat := handler.NewChatHandler(cfg, r)

	mux.Get("/health", health.Live)
	mux.Get("/ready", health.Ready)
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
	return s.http.Shutdown(ctx)
}
