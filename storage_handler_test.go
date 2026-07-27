package microscope

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStorageAPIReturnsUsage(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	hub.Record(context.Background(), Entry{Type: TypeMetric, Content: map[string]any{"name": "go.runtime"}})
	waitForStoredEntries(t, store, 1)

	mux := http.NewServeMux()
	(&Handler{Hub: hub}).RegisterRoutes(mux)

	request := httptest.NewRequest(http.MethodGet, "/microscope/api/storage", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected storage status 200, got %d: %s", response.Code, response.Body.String())
	}

	var usage StorageUsage
	if err := json.NewDecoder(response.Body).Decode(&usage); err != nil {
		t.Fatal(err)
	}
	if usage.EntryCount != 1 {
		t.Fatalf("expected entry_count 1, got %d", usage.EntryCount)
	}
	if usage.TotalMB != usage.EntriesMB+usage.SettingsMB+usage.MigrationsMB {
		t.Fatalf("expected total_mb to equal entries_mb + settings_mb + migrations_mb, got %+v", usage)
	}
}
