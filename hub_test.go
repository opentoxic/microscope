package microscope

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"
)

type memStore struct {
	mu       sync.Mutex
	entries  []Entry
	settings map[EntryType]bool
	options  map[string]json.RawMessage
}

func (m *memStore) Insert(_ context.Context, e Entry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, e)
	return nil
}

func (m *memStore) Get(_ context.Context, id string) (*Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.entries {
		if e.ID == id {
			copy := e
			return &copy, nil
		}
	}
	return nil, errNotFound
}

func (m *memStore) List(_ context.Context, f ListFilter) (ListResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var filtered []Entry
	for _, e := range m.entries {
		if f.Type != "" && e.Type != f.Type {
			continue
		}
		if f.RequestID != "" && e.RequestID != f.RequestID {
			continue
		}
		filtered = append(filtered, e)
	}
	return ListResult{Entries: filtered, Total: len(filtered)}, nil
}

func (m *memStore) ListByBatch(_ context.Context, batchID string) ([]Entry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []Entry
	for _, e := range m.entries {
		if e.BatchID == batchID {
			out = append(out, e)
		}
	}
	return out, nil
}

func (m *memStore) Prune(_ context.Context, olderThan time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []Entry
	var deleted int64
	for _, e := range m.entries {
		if e.CreatedAt.Before(olderThan) {
			deleted++
		} else {
			kept = append(kept, e)
		}
	}
	m.entries = kept
	return deleted, nil
}

func (m *memStore) ClearAll(_ context.Context) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := int64(len(m.entries))
	m.entries = nil
	return n, nil
}

func (m *memStore) ListTypeSettings(_ context.Context) ([]TypeSetting, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	counts := make(map[EntryType]int64)
	for _, entry := range m.entries {
		counts[entry.Type]++
	}
	out := make([]TypeSetting, 0, len(AllEntryTypes))
	for _, entryType := range AllEntryTypes {
		enabled := true
		if value, ok := m.settings[entryType]; ok {
			enabled = value
		}
		out = append(out, TypeSetting{Type: entryType, Enabled: enabled, Count: counts[entryType]})
	}
	return out, nil
}

func (m *memStore) StorageUsage(_ context.Context) (StorageUsage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	count := int64(len(m.entries))
	return StorageUsage{
		EntriesMB:        float64(count) * 0.01,
		EntriesDataMB:    float64(count) * 0.008,
		EntriesIndexesMB: float64(count) * 0.002,
		SettingsMB:       0.01,
		MigrationsMB:     0.03,
		TotalMB:          float64(count)*0.01 + 0.01 + 0.03,
		EntryCount:       count,
	}, nil
}

func (m *memStore) SetTypeEnabled(_ context.Context, entryType EntryType, enabled bool) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.settings == nil {
		m.settings = make(map[EntryType]bool)
	}
	m.settings[entryType] = enabled
	if enabled {
		return 0, nil
	}
	kept := m.entries[:0]
	var deleted int64
	for _, entry := range m.entries {
		if entry.Type == entryType {
			deleted++
			continue
		}
		kept = append(kept, entry)
	}
	m.entries = kept
	return deleted, nil
}

func (m *memStore) GetOption(_ context.Context, key string) (json.RawMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.options == nil {
		return nil, nil
	}
	value, ok := m.options[key]
	if !ok {
		return nil, nil
	}
	return append(json.RawMessage(nil), value...), nil
}

func (m *memStore) SetOption(_ context.Context, key string, value json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.options == nil {
		m.options = make(map[string]json.RawMessage)
	}
	m.options[key] = append(json.RawMessage(nil), value...)
	return nil
}

var errNotFound = errors.New("entry not found")

func TestHubRecord(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)

	ctx := WithBatchID(context.Background(), "batch-1")
	hub.Record(ctx, Entry{
		ID:      "entry-1",
		Type:    TypeRequest,
		Content: map[string]any{"path": "/health"},
	})

	deadline := time.Now().Add(2 * time.Second)
	for {
		store.mu.Lock()
		n := len(store.entries)
		store.mu.Unlock()
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for entry")
		}
		time.Sleep(10 * time.Millisecond)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if store.entries[0].BatchID != "batch-1" {
		t.Fatalf("expected batch-1, got %s", store.entries[0].BatchID)
	}
}

func TestRecordingPausedStopsRecord(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)

	hub.SetRecordingPaused(true)
	hub.Record(context.Background(), Entry{Type: TypeMetric, Content: map[string]any{"name": "blocked"}})
	time.Sleep(30 * time.Millisecond)

	store.mu.Lock()
	count := len(store.entries)
	store.mu.Unlock()
	if count != 0 {
		t.Fatalf("expected no entries while paused, got %d", count)
	}

	hub.SetRecordingPaused(false)
	hub.Record(context.Background(), Entry{Type: TypeMetric, Content: map[string]any{"name": "allowed"}})
	waitForStoredEntries(t, store, 1)
}

func TestEnabled(t *testing.T) {
	if !Enabled("development", Config{Enabled: true}) {
		t.Fatal("expected enabled in development")
	}
	if Enabled("production", Config{Enabled: true}) {
		t.Fatal("expected disabled in production")
	}
	if Enabled("development", Config{Enabled: false}) {
		t.Fatal("expected disabled when config disabled")
	}
}

func TestDisableTypePurgesAndStopsRecording(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)

	hub.Record(context.Background(), Entry{Type: TypeMetric, Content: map[string]any{"name": "before"}})
	waitForStoredEntries(t, store, 1)

	deleted, err := hub.SetTypeEnabled(context.Background(), TypeMetric, false)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("expected one deleted metric, got %d", deleted)
	}
	hub.Record(context.Background(), Entry{Type: TypeMetric, Content: map[string]any{"name": "after"}})
	time.Sleep(30 * time.Millisecond)

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.entries) != 0 {
		t.Fatalf("disabled signal recorded %d entries", len(store.entries))
	}
}

func TestReenableTypeResumesRecording(t *testing.T) {
	store := &memStore{}
	hub := NewWithStore(store, DefaultConfig(), nil)
	if _, err := hub.SetTypeEnabled(context.Background(), TypeQuery, false); err != nil {
		t.Fatal(err)
	}
	if _, err := hub.SetTypeEnabled(context.Background(), TypeQuery, true); err != nil {
		t.Fatal(err)
	}
	hub.Record(context.Background(), Entry{Type: TypeQuery, Content: map[string]any{"sql": "SELECT 1"}})
	waitForStoredEntries(t, store, 1)
}

func waitForStoredEntries(t *testing.T, store *memStore, expected int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		store.mu.Lock()
		count := len(store.entries)
		store.mu.Unlock()
		if count >= expected {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %d entries", expected)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
