package microscope

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("MICROSCOPE_ENABLED", "true")
	t.Setenv("MICROSCOPE_PATH", "/signals")
	t.Setenv("MICROSCOPE_RETENTION_HOURS", "48")
	t.Setenv("MICROSCOPE_MAX_BODY_BYTES", "8192")
	t.Setenv("MICROSCOPE_ALLOWED_ENVS", "development,staging")
	t.Setenv("MICROSCOPE_AUTO_MIGRATE", "false")

	cfg := ConfigFromEnv()
	if !cfg.Enabled {
		t.Fatal("expected enabled")
	}
	if cfg.Path != "/signals" {
		t.Fatalf("path %q", cfg.Path)
	}
	if cfg.RetentionHours != 48 {
		t.Fatalf("retention %d", cfg.RetentionHours)
	}
	if cfg.MaxBodyBytes != 8192 {
		t.Fatalf("max body %d", cfg.MaxBodyBytes)
	}
	if len(cfg.AllowedEnvs) != 2 || cfg.AllowedEnvs[1] != "staging" {
		t.Fatalf("allowed envs %v", cfg.AllowedEnvs)
	}
	if cfg.AutoMigrate {
		t.Fatal("expected auto migrate false")
	}
}

func TestMergeConfig(t *testing.T) {
	base := ConfigFromEnv()
	overrides := Config{
		Enabled:        false,
		Path:           "/custom",
		RetentionHours: 12,
		MaxBodyBytes:   1024,
	}
	merged := MergeConfig(base, overrides)
	if merged.Enabled {
		t.Fatal("expected disabled from override")
	}
	if merged.Path != "/custom" {
		t.Fatalf("path %q", merged.Path)
	}
}

func TestIntegrationInactive(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false
	integ := NewIntegration("development", cfg)
	if integ.Active() {
		t.Fatal("expected inactive")
	}
	if integ.QueryTracer() != nil {
		t.Fatal("expected nil tracer")
	}
	if integ.Bind(nil, nil) != nil {
		t.Fatal("expected nil hub")
	}
}

func TestIntegrationActiveWithoutPool(t *testing.T) {
	cfg := DefaultConfig()
	integ := NewIntegration("development", cfg)
	if !integ.Active() {
		t.Fatal("expected active")
	}
	if integ.QueryTracer() == nil {
		t.Fatal("expected tracer")
	}
}

func TestMigrationFS(t *testing.T) {
	files := MigrationFiles()
	if len(files) != 3 {
		t.Fatalf("expected 3 migration files, got %d", len(files))
	}
	for _, name := range files {
		data, err := MigrationFS().Open("migrations/" + name)
		if err != nil {
			t.Fatalf("open %s: %v", name, err)
		}
		_ = data.Close()
	}
}

func TestIntegrationHTTPMiddlewaresSkipAccessLog(t *testing.T) {
	store := &memStore{}
	cfg := DefaultConfig()
	hub := NewWithStore(store, cfg, nil)
	integ := &Integration{cfg: cfg, active: true, hub: hub}

	var logged int
	accessLog := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logged++
			next.ServeHTTP(w, r)
		})
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	integ.RegisterRoutes(mux)

	middlewares := integ.HTTPMiddlewares(HTTPOptions{
		AccessLog: accessLog,
	})
	handler := chainHandlers(mux, middlewares...)

	req := httptest.NewRequest(http.MethodGet, "/microscope/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if logged != 0 {
		t.Fatalf("expected no access log for microscope path, got %d", logged)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)
	if logged != 1 {
		t.Fatalf("expected access log for /health, got %d", logged)
	}
}

func TestWrapOTPNotifierRedacts(t *testing.T) {
	store := &memStore{}
	cfg := DefaultConfig()
	cfg.RedactSensitive = true
	hub := NewWithStore(store, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	inner := OTPNotifierFunc(func(ctx context.Context, kind, email, otp string) error {
		return nil
	})
	wrapped := WrapOTPNotifier(hub, &otpNotifierFuncAdapter{fn: inner})

	ctx := WithBatchID(context.Background(), "batch-otp")
	if err := wrapped.SendSignupOTP(ctx, "user@example.com", "123456"); err != nil {
		t.Fatal(err)
	}

	waitForEntries(t, store, 1)
	store.mu.Lock()
	defer store.mu.Unlock()
	entry := store.entries[0]
	if entry.Type != TypeNotification {
		t.Fatalf("type %s", entry.Type)
	}
	if entry.Content["otp"] != "[REDACTED]" {
		t.Fatalf("otp %v", entry.Content["otp"])
	}
}

func TestWrapOTPNotifierRecordsFullOTP(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))

	inner := OTPNotifierFunc(func(ctx context.Context, kind, email, otp string) error {
		return nil
	})
	wrapped := WrapOTPNotifier(hub, &otpNotifierFuncAdapter{fn: inner})

	ctx := WithBatchID(context.Background(), "batch-otp")
	if err := wrapped.SendSignupOTP(ctx, "user@example.com", "123456"); err != nil {
		t.Fatal(err)
	}

	waitForEntries(t, store, 1)
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.entries[0].Content["otp"] != "123456" {
		t.Fatalf("otp %v", store.entries[0].Content["otp"])
	}
}

func TestWrapEventPublisher(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), slog.New(slog.NewTextHandler(io.Discard, nil)))

	var published bool
	inner := EventPublisherFunc(func(ctx context.Context, eventType string, payload map[string]any) error {
		published = true
		return nil
	})
	wrapped := WrapEventPublisher(hub, inner)
	if err := wrapped.Publish(context.Background(), "user.created", map[string]any{"id": "u1"}); err != nil {
		t.Fatal(err)
	}
	if !published {
		t.Fatal("inner not called")
	}
	waitForEntries(t, store, 1)
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.entries[0].Type != TypeEvent {
		t.Fatalf("type %s", store.entries[0].Type)
	}
}

// Test helpers for decorator tests.
type OTPNotifierFunc func(ctx context.Context, kind, email, otp string) error

type otpNotifierFuncAdapter struct {
	fn OTPNotifierFunc
}

func (a *otpNotifierFuncAdapter) SendSignupOTP(ctx context.Context, email, otp string) error {
	return a.fn(ctx, "signup_otp", email, otp)
}

func (a *otpNotifierFuncAdapter) SendPasswordResetOTP(ctx context.Context, email, otp string) error {
	return a.fn(ctx, "password_reset_otp", email, otp)
}

func (a *otpNotifierFuncAdapter) SendEmailChangeOTP(ctx context.Context, email, otp string) error {
	return a.fn(ctx, "email_change_otp", email, otp)
}

type EventPublisherFunc func(ctx context.Context, eventType string, payload map[string]any) error

func (f EventPublisherFunc) Publish(ctx context.Context, eventType string, payload map[string]any) error {
	return f(ctx, eventType, payload)
}

func TestMain(m *testing.M) {
	_ = os.Setenv("MICROSCOPE_ENABLED", "")
	os.Exit(m.Run())
}

func chainHandlers(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
