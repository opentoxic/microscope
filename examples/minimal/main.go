// Minimal in-process microscope integration using the Setup API.
//
// Run from repo root:
//
//	APP_ENV=development DATABASE_URL=postgres://... go run ./examples/minimal
package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/opentoxic/microscope"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ms, err := microscope.Setup(ctx, microscope.SetupOptions{
		AppEnv: envOr("APP_ENV", "development"),
		DSN:    dsn,
		Logger: log,
	})
	if err != nil {
		log.Error("microscope setup failed", "error", err)
		os.Exit(1)
	}
	defer ms.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	ms.RegisterRoutes(mux)

	addr := envOr("HTTP_ADDR", ":8080")
	handler := ms.HTTPHandler(mux, microscope.HTTPOptions{
		AccessLog: microscope.SimpleAccessLog(ms.Logger),
	})

	srv := &http.Server{Addr: addr, Handler: handler}
	go func() {
		log.Info("listening", "addr", addr, "microscope", ms.Config().Path, "active", ms.Active())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	<-stop.Done()

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
