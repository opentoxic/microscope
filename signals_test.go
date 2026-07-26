package microscope

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTypedSignalsAreRecordedAndPublished(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	stream, unsubscribe := hub.Subscribe(16)
	defer unsubscribe()
	ctx := WithBatchID(context.Background(), "batch-signals")

	hub.RecordCache(ctx, "get", "session:1", true, time.Millisecond, nil)
	hub.RecordRedis(ctx, "GET", time.Millisecond, nil, nil)
	hub.RecordJob(ctx, "send-email", "default", "processed", time.Millisecond, nil)
	hub.RecordSchedule(ctx, "prune", "finished", time.Millisecond, nil)
	hub.RecordMail(ctx, "Welcome", []string{"user@example.com"}, "sent", time.Millisecond, nil)
	hub.RecordWebSocket(ctx, "message", "updates", "outgoing", 42, nil)
	hub.RecordPerformance(ctx, "password.hash", time.Millisecond, nil)
	hub.RecordMetric(ctx, "workers", 4, "count", nil)
	hub.RecordCustom(ctx, "checkpoint", map[string]any{"token": "secret"})
	hub.RecordTopic(ctx, "identity.events", "produce", time.Millisecond, map[string]any{"message_count": 1})

	waitForEntries(t, store, 10)

	seen := make(map[EntryType]bool)
	for range 10 {
		select {
		case entry := <-stream:
			seen[entry.Type] = true
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for published signal")
		}
	}
	for _, entryType := range []EntryType{
		TypeCache, TypeRedis, TypeJob, TypeSchedule, TypeMail, TypeWebSocket,
		TypePerformance, TypeMetric, TypeCustom,
		TypeTopic,
	} {
		if !seen[entryType] {
			t.Fatalf("expected published type %s", entryType)
		}
	}
}

func TestLiveEntryStreamPublishesPersistedEntryOnce(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	handler := &Handler{Hub: hub}
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	server := httptest.NewServer(mux)
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/microscope/api/stream", nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	hub.RecordCustom(context.Background(), "live-test", nil)
	scanner := bufio.NewScanner(response.Body)
	var dataLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataLines = append(dataLines, line)
			break
		}
	}
	if len(dataLines) != 1 || !strings.Contains(dataLines[0], `"name":"live-test"`) {
		t.Fatalf("expected one live custom entry, got %v", dataLines)
	}
}

func TestCreateCustomEntryEndpoint(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	handler := &Handler{Hub: hub}
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]any{
		"name":    "release checkpoint",
		"content": map[string]any{"commit": "abc123"},
	})
	request := httptest.NewRequest(http.MethodPost, "/microscope/api/entries", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", response.Code, response.Body.String())
	}
	waitForEntries(t, store, 1)
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.entries[0].Type != TypeCustom || store.entries[0].Content["name"] != "release checkpoint" {
		t.Fatalf("unexpected custom entry: %#v", store.entries[0])
	}
}
