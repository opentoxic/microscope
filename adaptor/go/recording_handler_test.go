package microscope

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecordingAPIRoundTrip(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	mux := http.NewServeMux()
	(&Handler{Hub: hub}).RegisterRoutes(mux)

	getReq := httptest.NewRequest(http.MethodGet, "/microscope/api/recording", nil)
	getRes := httptest.NewRecorder()
	mux.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("GET recording status %d: %s", getRes.Code, getRes.Body.String())
	}
	var initial struct {
		Paused bool `json:"paused"`
	}
	if err := json.NewDecoder(getRes.Body).Decode(&initial); err != nil {
		t.Fatal(err)
	}
	if initial.Paused {
		t.Fatal("expected recording to start unpaused")
	}

	body, _ := json.Marshal(map[string]bool{"paused": true})
	putReq := httptest.NewRequest(http.MethodPut, "/microscope/api/recording", bytes.NewReader(body))
	putReq.Header.Set("Content-Type", "application/json")
	putRes := httptest.NewRecorder()
	mux.ServeHTTP(putRes, putReq)
	if putRes.Code != http.StatusOK {
		t.Fatalf("PUT recording status %d: %s", putRes.Code, putRes.Body.String())
	}
	var paused struct {
		Paused bool `json:"paused"`
	}
	if err := json.NewDecoder(putRes.Body).Decode(&paused); err != nil {
		t.Fatal(err)
	}
	if !paused.Paused {
		t.Fatal("expected paused true after PUT")
	}

	hub.Record(context.Background(), Entry{Type: TypeMetric, Content: map[string]any{"name": "blocked"}})
	waitForStoredEntries(t, store, 0)

	unpauseBody, _ := json.Marshal(map[string]bool{"paused": false})
	unpauseReq := httptest.NewRequest(http.MethodPut, "/microscope/api/recording", bytes.NewReader(unpauseBody))
	unpauseReq.Header.Set("Content-Type", "application/json")
	unpauseRes := httptest.NewRecorder()
	mux.ServeHTTP(unpauseRes, unpauseReq)
	if unpauseRes.Code != http.StatusOK {
		t.Fatalf("PUT resume status %d: %s", unpauseRes.Code, unpauseRes.Body.String())
	}

	hub.Record(context.Background(), Entry{Type: TypeMetric, Content: map[string]any{"name": "allowed"}})
	waitForStoredEntries(t, store, 1)
}
