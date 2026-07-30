package microscope

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type testRequestMetaKey struct{}

func TestMiddlewareSkipsMicroscopePaths(t *testing.T) {
	store := &memStore{}
	cfg := DefaultConfig()
	hub := NewWithStore(store, cfg, nil)

	var recorded int
	mux := http.NewServeMux()
	mux.HandleFunc("GET /microscope", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	bridge := BridgeMiddleware(func(ctx context.Context) RequestMeta {
		meta, _ := ctx.Value(testRequestMetaKey{}).(RequestMeta)
		return meta
	})
	identity := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			meta := RequestMeta{
				RequestID:     "request-test",
				CorrelationID: "correlation-test",
				IPAddress:     "127.0.0.1",
				UserAgent:     "microscope-test",
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), testRequestMetaKey{}, meta)))
		})
	}
	handler := identity(bridge(hub.Middleware()(mux)))

	req := httptest.NewRequest(http.MethodGet, "/microscope", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	waitForEntries(t, store, 1)
	store.mu.Lock()
	recorded = len(store.entries)
	store.mu.Unlock()
	if recorded != 1 {
		t.Fatalf("expected 1 entry, got %d", recorded)
	}
	if store.entries[0].Type != TypeRequest {
		t.Fatalf("expected request type, got %s", store.entries[0].Type)
	}
	if store.entries[0].Content["path"] != "/health" {
		t.Fatalf("unexpected path %v", store.entries[0].Content["path"])
	}
}

func TestHandlerListAndGet(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	now := time.Now().UTC()

	_ = store.Insert(context.Background(), Entry{
		ID: "e1", BatchID: "b1", Type: TypeRequest,
		Content:   map[string]any{"method": "GET", "path": "/health", "status": 200},
		CreatedAt: now,
	})
	_ = store.Insert(context.Background(), Entry{
		ID: "e2", BatchID: "b1", Type: TypeQuery,
		Content:   map[string]any{"sql": "SELECT 1"},
		CreatedAt: now,
	})

	h := &Handler{Hub: hub}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/microscope/api/entries", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status %d body %s", rr.Code, rr.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodGet, "/microscope/api/entries/e1", nil)
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("get status %d body %s", rr2.Code, rr2.Body.String())
	}
	var detail EntryDetailResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if detail.Entry.ID != "e1" {
		t.Fatalf("expected entry e1, got %s", detail.Entry.ID)
	}
	if len(detail.ContentTabs) == 0 {
		t.Fatalf("expected content tabs")
	}
}

func TestNotificationRecordsOTPWhenRedactionEnabled(t *testing.T) {
	store := &memStore{}
	cfg := DefaultConfig()
	cfg.RedactSensitive = BoolPtr(true)
	hub := NewWithStore(store, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx := WithBatchID(context.Background(), "batch-1")
	hub.RecordNotification(ctx, "signup_otp", map[string]any{
		"email": "user@example.com",
		"otp":   hub.SanitizeOTP("123456"),
	})

	waitForEntries(t, store, 1)
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.entries[0].Content["otp"] != "[REDACTED]" {
		t.Fatalf("otp should be redacted, got %v", store.entries[0].Content["otp"])
	}
}

func TestDashboardServesSPA(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	h := &Handler{Hub: hub}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/microscope/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("spa status %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `<div id="app">`) && !strings.Contains(body, `<div id="app"></div>`) {
		t.Fatalf("expected vue mount point in index.html")
	}
}

func TestEntryDetailPage(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	now := time.Now().UTC()

	_ = store.Insert(context.Background(), Entry{
		ID: "e1", BatchID: "b1", Type: TypeRequest,
		Content:   map[string]any{"method": "GET", "path": "/health", "status": 200},
		CreatedAt: now,
	})
	_ = store.Insert(context.Background(), Entry{
		ID: "e2", BatchID: "b1", Type: TypeQuery,
		Content:   map[string]any{"sql": "SELECT 1", "duration_ms": 5},
		CreatedAt: now,
	})
	_ = store.Insert(context.Background(), Entry{
		ID: "e3", BatchID: "b1", Type: TypeLog,
		Content:   map[string]any{"level": "info", "message": "ok"},
		CreatedAt: now,
	})

	h := &Handler{Hub: hub}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/microscope/entries/e1", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("detail spa status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `<div id="app"`) {
		t.Fatalf("expected spa index for detail route")
	}

	reqAPI := httptest.NewRequest(http.MethodGet, "/microscope/api/entries/e1", nil)
	rrAPI := httptest.NewRecorder()
	mux.ServeHTTP(rrAPI, reqAPI)
	var detail EntryDetailResponse
	if err := json.Unmarshal(rrAPI.Body.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(detail.Entry.Content["path"].(string), "/health") {
		t.Fatalf("expected /health in api response")
	}
	if len(detail.BatchGroups) < 2 {
		t.Fatalf("expected related batch groups")
	}
}

func TestSettingsPageServesSPA(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	h := &Handler{Hub: hub}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/microscope/settings", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("settings spa status %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `<div id="app"`) {
		t.Fatalf("expected spa index for settings route")
	}
}

func TestHandlerPruneClearsAll(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	now := time.Now().UTC()
	_ = store.Insert(context.Background(), Entry{ID: "e1", BatchID: "b1", Type: TypeRequest, CreatedAt: now})
	_ = store.Insert(context.Background(), Entry{ID: "e2", BatchID: "b2", Type: TypeQuery, CreatedAt: now})

	h := &Handler{Hub: hub}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/microscope/api/prune", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("prune status %d", rr.Code)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.entries) != 0 {
		t.Fatalf("expected all entries cleared, got %d", len(store.entries))
	}
}

func waitForEntries(t *testing.T, store *memStore, n int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		store.mu.Lock()
		count := len(store.entries)
		store.mu.Unlock()
		if count >= n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d entries, got %d", n, count)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
