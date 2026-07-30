package microscope

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRedactMap(t *testing.T) {
	m := map[string]any{
		"email":    "user@example.com",
		"password": "secret123",
		"otp":      "123456",
		"nested": map[string]any{
			"refresh_token": "abc",
			"name":          "test",
		},
	}
	out := RedactMap(m)
	if out["password"] != "[REDACTED]" {
		t.Fatalf("expected password redacted, got %v", out["password"])
	}
	if out["otp"] != "[REDACTED]" {
		t.Fatalf("expected otp redacted, got %v", out["otp"])
	}
	if out["email"] != "user@example.com" {
		t.Fatalf("expected email preserved, got %v", out["email"])
	}
	nested := out["nested"].(map[string]any)
	if nested["refresh_token"] != "[REDACTED]" {
		t.Fatalf("expected nested refresh_token redacted")
	}
	if nested["name"] != "test" {
		t.Fatalf("expected nested name preserved")
	}
}

func TestRedactHeaders(t *testing.T) {
	headers := map[string][]string{
		"Authorization": {"Bearer secret"},
		"Content-Type":  {"application/json"},
	}
	out := RedactHeaders(headers)
	if out["Authorization"][0] != "[REDACTED]" {
		t.Fatalf("expected authorization redacted")
	}
	if out["Content-Type"][0] != "application/json" {
		t.Fatalf("expected content-type preserved")
	}
}

func TestRedactJSON(t *testing.T) {
	body := []byte(`{"email":"a@b.com","password":"pw"}`)
	out := RedactJSON(body)
	if out == "" {
		t.Fatal("expected non-empty output")
	}
	if contains(out, "pw") {
		t.Fatalf("password should be redacted in %s", out)
	}
}

func TestHubSanitizeMapModes(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	input := map[string]any{
		"password": "secret",
		"email":    "user@example.com",
	}

	full := hub.SanitizeMap(input)
	if full["password"] != "secret" {
		t.Fatalf("expected full password, got %v", full["password"])
	}

	cfg := DefaultConfig()
	cfg.RedactSensitive = BoolPtr(true)
	redactingHub := NewWithStore(store, cfg, nil)
	redacted := redactingHub.SanitizeMap(input)
	if redacted["password"] != "[REDACTED]" {
		t.Fatalf("expected redacted password, got %v", redacted["password"])
	}
}

func TestHubSanitizeOTP(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	if hub.SanitizeOTP("123456") != "123456" {
		t.Fatal("expected full otp by default")
	}

	cfg := DefaultConfig()
	cfg.RedactSensitive = BoolPtr(true)
	redactingHub := NewWithStore(store, cfg, nil)
	if redactingHub.SanitizeOTP("123456") != "[REDACTED]" {
		t.Fatal("expected redacted otp")
	}
}

func TestHandlerRedactionAPI(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	h := &Handler{Hub: hub}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/microscope/api/redaction", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("get status %d", rr.Code)
	}

	body := `{"enabled":true}`
	req2 := httptest.NewRequest(http.MethodPut, "/microscope/api/redaction", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	mux.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("put status %d body %s", rr2.Code, rr2.Body.String())
	}
	if !hub.RedactSensitive() {
		t.Fatal("expected redaction enabled")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
