package microscope

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSettingsAPIListsAndDisablesSignal(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	hub.Record(context.Background(), Entry{Type: TypeMetric, Content: map[string]any{"name": "go.runtime"}})
	waitForStoredEntries(t, store, 1)

	mux := http.NewServeMux()
	(&Handler{Hub: hub}).RegisterRoutes(mux)

	listRequest := httptest.NewRequest(http.MethodGet, "/microscope/api/settings", nil)
	listResponse := httptest.NewRecorder()
	mux.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected settings list status 200, got %d", listResponse.Code)
	}

	body := bytes.NewBufferString(`{"enabled":false}`)
	updateRequest := httptest.NewRequest(http.MethodPut, "/microscope/api/settings/metric", body)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateResponse := httptest.NewRecorder()
	mux.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("expected settings update status 200, got %d: %s", updateResponse.Code, updateResponse.Body.String())
	}

	var result struct {
		Enabled bool  `json:"enabled"`
		Deleted int64 `json:"deleted"`
	}
	if err := json.NewDecoder(updateResponse.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Enabled || result.Deleted != 1 {
		t.Fatalf("expected disabled setting with one deleted row, got %+v", result)
	}
}

func TestSettingsAPIRejectsUnknownSignal(t *testing.T) {
	hub := NewWithStore(&memStore{}, DefaultConfig(), nil)
	mux := http.NewServeMux()
	(&Handler{Hub: hub}).RegisterRoutes(mux)

	request := httptest.NewRequest(http.MethodPut, "/microscope/api/settings/not-real", bytes.NewBufferString(`{"enabled":false}`))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", response.Code)
	}
}
