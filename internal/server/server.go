package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"signalhub/internal/config"
	"signalhub/internal/domain"
	"signalhub/internal/server/handler"
	"signalhub/internal/server/middleware"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
	repo       domain.Repository
}

func New(cfg *config.Config, logger *slog.Logger, repo domain.Repository) *Server {
	mux := http.NewServeMux()

	webhookHandler := handler.NewWebhookHandler(repo, logger)

	// Public routes
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// In a real app, check DB connection here
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	})

	mux.Handle("/metrics", promhttp.Handler())

	// Protected routes
	webhookMux := http.NewServeMux()
	webhookMux.HandleFunc("/webhooks/generic", webhookHandler.HandleGeneric)
	webhookMux.HandleFunc("/webhooks/prometheus", webhookHandler.HandlePrometheus)

	// Chain middlewares
	// 1. Request ID
	// 2. Payload limit (1MB default)
	// 3. Auth
	var handler http.Handler = webhookMux
	handler = middleware.Auth(cfg.Auth.WebhookToken)(handler)
	handler = middleware.PayloadLimit(1024 * 1024)(handler)
	handler = middleware.RequestID(handler)

	mux.Handle("/webhooks/", handler)

	return &Server{
		httpServer: &http.Server{
			Addr:         cfg.Server.Address,
			Handler:      mux,
			ReadTimeout:  cfg.Server.ReadTimeout,
			WriteTimeout: cfg.Server.WriteTimeout,
		},
		logger: logger,
		repo:   repo,
	}
}

func (s *Server) Start() error {
	s.logger.Info("Starting server", "address", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server")
	return s.httpServer.Shutdown(ctx)
}
