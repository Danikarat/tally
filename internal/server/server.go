package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Danikarat/tally/internal/auth"
	"github.com/Danikarat/tally/internal/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Server represents the HTTP application server.
type Server struct {
	cfg        *config.Config
	logger     *slog.Logger
	router     *chi.Mux
	db         *pgxpool.Pool
	httpServer *http.Server
}

// New creates and configures a new Server instance.
func New(cfg *config.Config, logger *slog.Logger, db *pgxpool.Pool) *Server {
	s := &Server{
		cfg:    cfg,
		logger: logger,
		router: chi.NewRouter(),
		db:     db,
	}

	s.setupMiddleware()
	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      s.router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	return s
}

// setupMiddleware attaches standard HTTP middlewares.
func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(s.slogMiddleware())
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))
}

// slogMiddleware logs HTTP requests using standard slog logger.
func (s *Server) slogMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				s.logger.Info("http request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"duration_ms", time.Since(start).Milliseconds(),
					"req_id", middleware.GetReqID(r.Context()),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}

// setupRoutes registers application endpoints.
func (s *Server) setupRoutes() {
	s.router.Get("/healthz", s.handleHealthCheck)

	if s.db != nil {
		authService := auth.NewService(s.db, s.cfg.Auth)
		authHandler := auth.NewHandler(authService)
		authHandler.RegisterRoutes(s.router)
	}
}

// handleHealthCheck returns 200 OK with server status.
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	resp := map[string]string{
		"status":      "ok",
		"environment": s.cfg.Environment,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// Start initiates HTTP listening.
func (s *Server) Start() error {
	s.logger.Info("starting http server", "addr", s.httpServer.Addr, "env", s.cfg.Environment)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("http server failure: %w", err)
	}
	return nil
}

// Shutdown gracefully terminates the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down http server")
	return s.httpServer.Shutdown(ctx)
}

// Router exposes the underlying chi.Mux router (primarily for testing).
func (s *Server) Router() *chi.Mux {
	return s.router
}
