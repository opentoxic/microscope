package microscope

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestEnabledAllowedEnvs(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true

	if !Enabled("development", cfg) {
		t.Fatal("expected development")
	}
	if !Enabled("local", cfg) {
		t.Fatal("expected local")
	}
	if Enabled("production", cfg) {
		t.Fatal("expected production disabled")
	}

	cfg.AllowedEnvs = []string{"staging"}
	if !Enabled("staging", cfg) {
		t.Fatal("expected custom staging env")
	}
}

func TestConfigurePoolAttachesTracer(t *testing.T) {
	cfg := DefaultConfig()
	integ := NewIntegration("development", cfg)
	poolCfg, err := pgxpool.ParseConfig("postgres://user:pass@localhost:5432/db")
	if err != nil {
		t.Fatal(err)
	}
	integ.ConfigurePool(poolCfg)
	if poolCfg.ConnConfig.Tracer == nil {
		t.Fatal("expected tracer attached")
	}

	inactive := NewIntegration("production", cfg)
	inactive.ConfigurePool(poolCfg)
}

func TestSetupRequiresDSN(t *testing.T) {
	_, err := Setup(context.Background(), SetupOptions{AppEnv: "development"})
	if err == nil {
		t.Fatal("expected error for missing DSN")
	}
}

func TestSetupInactive(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false
	integ := NewIntegration("development", cfg)
	if integ.Active() {
		t.Fatal("expected inactive setup integration")
	}
}

func TestHTTPHandlerSetsRequestID(t *testing.T) {
	store := &memStore{}
	cfg := DefaultConfig()
	hub := NewWithStore(store, cfg, nil)
	integ := &Integration{cfg: cfg, active: true, hub: hub}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api", func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) == "" {
			t.Fatal("expected request id in context")
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := integ.HTTPHandler(mux, HTTPOptions{})
	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID response header")
	}
	waitForEntries(t, store, 1)
}

func TestWrapPublishFunc(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	integ := &Integration{cfg: DefaultConfig(), active: true, hub: hub}

	var called bool
	wrapped := integ.WrapPublishFunc(func(ctx context.Context, eventType string, payload map[string]any) error {
		called = true
		return nil
	})
	if err := wrapped(context.Background(), "user.created", map[string]any{"id": "1"}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("inner not called")
	}
	waitForEntries(t, store, 1)
}
