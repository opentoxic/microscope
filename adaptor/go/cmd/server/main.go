// Command server runs microscope as a standalone HTTP service, so any stack
// (Go, Python, TypeScript, ...) can record entries through its HTTP API
// instead of importing the Go package directly.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/opentoxic/microscope/adaptor/go"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dbURL := os.Getenv("MICROSCOPE_DATABASE_URL")
	if dbURL == "" {
		logger.Error("MICROSCOPE_DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		logger.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	cfg := microscope.DefaultConfig()
	cfg.Path = envOr("MICROSCOPE_PATH", cfg.Path)
	cfg.RetentionHours = envIntOr("MICROSCOPE_RETENTION_HOURS", cfg.RetentionHours)
	cfg.MaxBodyBytes = envIntOr("MICROSCOPE_MAX_BODY_BYTES", cfg.MaxBodyBytes)

	hub := microscope.New(pool, cfg, logger)

	mux := http.NewServeMux()
	(&microscope.Handler{Hub: hub}).RegisterRoutes(mux)

	addr := envOr("MICROSCOPE_ADDR", ":8093")
	srv := &http.Server{
		Addr:              addr,
		Handler:           hub.Middleware()(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("microscope listening", "addr", addr, "path", cfg.Path)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("serve", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
