package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Danikarat/tally/internal/config"
	"github.com/Danikarat/tally/internal/database"
	_ "github.com/Danikarat/tally/internal/parsers/mtn"
	"github.com/Danikarat/tally/internal/server"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	var logHandler slog.Handler
	if cfg.Environment == "development" {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	logger.Info("bootstrapping tally api",
		"env", cfg.Environment,
		"port", cfg.Port,
	)

	// Context for graceful shutdown on SIGINT or SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Connect to database pool
	dbPool, err := database.NewPool(ctx, cfg.DB.DSN())
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbPool.Close()

	// Apply database migrations on boot
	if err := database.MigrateUp(cfg.DB.DSN()); err != nil {
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	srv := server.New(cfg, logger, dbPool)

	serverErrors := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server startup failed: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received, commencing graceful shutdown...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server graceful shutdown failed: %w", err)
	}

	logger.Info("tally api stopped cleanly")
	return nil
}
