package microscope

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var silentLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// Hub records and serves microscope entries.
type Hub struct {
	store  Store
	cfg    Config
	log    *slog.Logger
	stopCh chan struct{}
	wg     sync.WaitGroup

	queryTracer *QueryTracer
	subsMu      sync.RWMutex
	subscribers map[chan Entry]struct{}
	controlSubs map[chan ControlEvent]struct{}
	settingsMu  sync.RWMutex
	enabled     map[EntryType]bool
}

// New creates a Hub backed by PostgreSQL.
func New(pool *pgxpool.Pool, cfg Config, _ *slog.Logger) *Hub {
	h := &Hub{
		store:       NewPostgresStore(pool),
		cfg:         cfg,
		log:         silentLogger,
		stopCh:      make(chan struct{}),
		subscribers: make(map[chan Entry]struct{}),
		controlSubs: make(map[chan ControlEvent]struct{}),
		enabled:     defaultTypeSettings(),
	}
	h.loadTypeSettings()
	h.startPruner()
	h.startRuntimeSampler()
	return h
}

func (h *Hub) startRuntimeSampler() {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-h.stopCh:
				return
			case <-ticker.C:
				if !h.TypeEnabled(TypeMetric) {
					continue
				}
				var memory runtime.MemStats
				runtime.ReadMemStats(&memory)
				entryID := newID()
				entry := Entry{
					ID:      entryID,
					BatchID: entryID,
					Type:    TypeMetric,
					Tags:    []string{"metric:go.runtime"},
					Content: map[string]any{
						"name":         "go.runtime",
						"value":        runtime.NumGoroutine(),
						"unit":         "goroutines",
						"goroutines":   runtime.NumGoroutine(),
						"heap_mb":      float64(memory.HeapAlloc) / 1024 / 1024,
						"heap_objects": memory.HeapObjects,
						"gc_cycles":    memory.NumGC,
					},
					CreatedAt: time.Now().UTC(),
				}
				insertCtx, cancel := context.WithTimeout(WithoutTrace(context.Background()), 5*time.Second)
				h.settingsMu.RLock()
				if h.enabled[TypeMetric] {
					if err := h.store.Insert(insertCtx, entry); err == nil {
						h.publish(entry)
					}
				}
				h.settingsMu.RUnlock()
				cancel()
			}
		}
	}()
}

// NewWithTracer creates a Hub and binds it to an existing query tracer.
func NewWithTracer(pool *pgxpool.Pool, cfg Config, log *slog.Logger, tracer *QueryTracer) *Hub {
	h := New(pool, cfg, log)
	if tracer != nil {
		tracer.Bind(h)
		h.queryTracer = tracer
	}
	return h
}

// NewWithStore creates a Hub with a custom store (for tests).
func NewWithStore(store Store, cfg Config, _ *slog.Logger) *Hub {
	h := &Hub{
		store:       store,
		cfg:         cfg,
		log:         silentLogger,
		stopCh:      make(chan struct{}),
		subscribers: make(map[chan Entry]struct{}),
		controlSubs: make(map[chan ControlEvent]struct{}),
		enabled:     defaultTypeSettings(),
	}
	h.loadTypeSettings()
	return h
}

// QueryTracer returns the bound query tracer, if any.
func (h *Hub) QueryTracer() *QueryTracer {
	return h.queryTracer
}

// Config returns the hub configuration.
func (h *Hub) Config() Config {
	return h.cfg
}

// Record persists an entry asynchronously.
func (h *Hub) Record(ctx context.Context, e Entry) {
	if !h.TypeEnabled(e.Type) {
		return
	}
	if e.ID == "" {
		e.ID = newID()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	if e.BatchID == "" {
		e.BatchID = BatchIDFromContext(ctx)
	}
	if e.BatchID == "" {
		e.BatchID = e.ID
	}

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.settingsMu.RLock()
		defer h.settingsMu.RUnlock()
		if !h.enabled[e.Type] {
			return
		}
		insertCtx, cancel := context.WithTimeout(WithoutTrace(context.Background()), 10*time.Second)
		defer cancel()
		if err := h.store.Insert(insertCtx, e); err != nil {
			return // internal microscope errors stay in the dashboard only
		}
		h.publish(e)
	}()
}

func defaultTypeSettings() map[EntryType]bool {
	settings := make(map[EntryType]bool, len(AllEntryTypes))
	for _, entryType := range AllEntryTypes {
		settings[entryType] = true
	}
	return settings
}

func (h *Hub) loadTypeSettings() {
	ctx, cancel := context.WithTimeout(WithoutTrace(context.Background()), 5*time.Second)
	defer cancel()
	settings, err := h.store.ListTypeSettings(ctx)
	if err != nil {
		return
	}
	h.settingsMu.Lock()
	defer h.settingsMu.Unlock()
	for _, setting := range settings {
		h.enabled[setting.Type] = setting.Enabled
	}
}

// TypeEnabled reports whether a signal may be persisted.
func (h *Hub) TypeEnabled(entryType EntryType) bool {
	if !ValidEntryType(entryType) {
		return false
	}
	h.settingsMu.RLock()
	defer h.settingsMu.RUnlock()
	return h.enabled[entryType]
}

// TypeSettings returns all configured signal settings and their stored counts.
func (h *Hub) TypeSettings(ctx context.Context) ([]TypeSetting, error) {
	return h.store.ListTypeSettings(WithoutTrace(ctx))
}

// SetTypeEnabled atomically persists a signal state and purges its records when disabled.
func (h *Hub) SetTypeEnabled(ctx context.Context, entryType EntryType, enabled bool) (int64, error) {
	if !ValidEntryType(entryType) {
		return 0, fmt.Errorf("unknown signal type %q", entryType)
	}

	// Closing the gate before deletion prevents queued async writes from restoring purged data.
	h.settingsMu.Lock()
	defer h.settingsMu.Unlock()
	previous := h.enabled[entryType]
	h.enabled[entryType] = enabled

	deleted, err := h.store.SetTypeEnabled(WithoutTrace(ctx), entryType, enabled)
	if err != nil {
		h.enabled[entryType] = previous
		return 0, err
	}
	h.publishControl(ControlEvent{Action: "signal-setting", Type: entryType, Deleted: deleted})
	return deleted, nil
}

// Subscribe returns a live stream of newly persisted entries.
func (h *Hub) Subscribe(buffer int) (<-chan Entry, func()) {
	if buffer <= 0 {
		buffer = 32
	}
	ch := make(chan Entry, buffer)
	h.subsMu.Lock()
	h.subscribers[ch] = struct{}{}
	h.subsMu.Unlock()
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.subsMu.Lock()
			if _, ok := h.subscribers[ch]; ok {
				delete(h.subscribers, ch)
				close(ch)
			}
			h.subsMu.Unlock()
		})
	}
}

// ControlEvent describes a live mutation that affects already-rendered entries.
type ControlEvent struct {
	Action  string    `json:"action"`
	Type    EntryType `json:"type,omitempty"`
	Deleted int64     `json:"deleted"`
}

// SubscribeControls returns a live stream of prune and recording-policy mutations.
func (h *Hub) SubscribeControls(buffer int) (<-chan ControlEvent, func()) {
	if buffer <= 0 {
		buffer = 8
	}
	ch := make(chan ControlEvent, buffer)
	h.subsMu.Lock()
	h.controlSubs[ch] = struct{}{}
	h.subsMu.Unlock()
	var once sync.Once
	return ch, func() {
		once.Do(func() {
			h.subsMu.Lock()
			if _, ok := h.controlSubs[ch]; ok {
				delete(h.controlSubs, ch)
				close(ch)
			}
			h.subsMu.Unlock()
		})
	}
}

func (h *Hub) publish(entry Entry) {
	h.subsMu.RLock()
	defer h.subsMu.RUnlock()
	for subscriber := range h.subscribers {
		select {
		case subscriber <- entry:
		default:
			// A slow dashboard never blocks application recording.
		}
	}
}

func (h *Hub) publishControl(event ControlEvent) {
	h.subsMu.RLock()
	defer h.subsMu.RUnlock()
	for subscriber := range h.controlSubs {
		select {
		case subscriber <- event:
		default:
		}
	}
}

// Store exposes the underlying store for handlers.
func (h *Hub) Store() Store {
	return h.store
}

// Close stops background workers and waits for pending writes.
func (h *Hub) Close() {
	close(h.stopCh)
	h.wg.Wait()
	h.subsMu.Lock()
	for subscriber := range h.subscribers {
		close(subscriber)
		delete(h.subscribers, subscriber)
	}
	for subscriber := range h.controlSubs {
		close(subscriber)
		delete(h.controlSubs, subscriber)
	}
	h.subsMu.Unlock()
}

func (h *Hub) startPruner() {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-h.stopCh:
				return
			case <-ticker.C:
				h.pruneOnce()
			}
		}
	}()
}

func (h *Hub) pruneOnce() int64 {
	ctx, cancel := context.WithTimeout(WithoutTrace(context.Background()), 30*time.Second)
	defer cancel()
	cutoff := time.Now().UTC().Add(-h.cfg.retention())
	n, err := h.store.Prune(ctx, cutoff)
	if err != nil {
		return 0
	}
	return n
}

// Prune removes entries older than the configured retention period.
func (h *Hub) Prune() int64 {
	return h.pruneOnce()
}
