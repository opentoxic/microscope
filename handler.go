package microscope

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Handler serves the microscope dashboard and API.
type Handler struct {
	Hub *Hub
}

// RegisterRoutes mounts microscope routes on mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	prefix := h.Hub.cfg.pathPrefix()
	mux.HandleFunc("GET "+prefix+"/api/entries", h.listEntries)
	mux.HandleFunc("POST "+prefix+"/api/entries", h.createCustomEntry)
	mux.HandleFunc("GET "+prefix+"/api/entries/{id}", h.getEntry)
	mux.HandleFunc("GET "+prefix+"/api/stream", h.streamEntries)
	mux.HandleFunc("POST "+prefix+"/api/prune", h.pruneEntries)
	mux.HandleFunc("GET "+prefix+"/api/storage", h.getStorageUsage)
	mux.HandleFunc("GET "+prefix+"/api/recording", h.getRecordingState)
	mux.HandleFunc("PUT "+prefix+"/api/recording", h.setRecordingState)
	mux.HandleFunc("GET "+prefix+"/api/settings", h.listSettings)
	mux.HandleFunc("PUT "+prefix+"/api/settings/{type}", h.updateSetting)
	mux.HandleFunc("POST "+prefix+"/api/insights/analyze", h.analyzeInsights)
	mux.HandleFunc("POST "+prefix+"/api/insights/models", h.listLLMModels)
	h.registerSPARoutes(mux, prefix)
}

func (h *Handler) createCustomEntry(w http.ResponseWriter, r *http.Request) {
	if h.Hub.RecordingPaused() {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "recording is paused"})
		return
	}
	if !h.Hub.TypeEnabled(TypeCustom) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "custom events are disabled in settings"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var input struct {
		Name    string         `json:"name"`
		Content map[string]any `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" || len(input.Name) > 120 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "name must contain 1 to 120 characters"})
		return
	}
	entryID := newID()
	content := cloneContent(input.Content)
	content["name"] = input.Name
	h.Hub.Record(r.Context(), Entry{
		ID:        entryID,
		Type:      TypeCustom,
		Tags:      []string{"custom:" + input.Name},
		Content:   RedactMap(content),
		CreatedAt: time.Now().UTC(),
	})
	writeJSON(w, http.StatusAccepted, map[string]string{"id": entryID})
}

func (h *Handler) getStorageUsage(w http.ResponseWriter, r *http.Request) {
	usage, err := h.Hub.store.StorageUsage(WithoutTrace(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, usage)
}

func (h *Handler) getRecordingState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"paused": h.Hub.RecordingPaused()})
}

func (h *Handler) setRecordingState(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var input struct {
		Paused *bool `json:"paused"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Paused == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "paused must be a boolean"})
		return
	}
	h.Hub.SetRecordingPaused(*input.Paused)
	writeJSON(w, http.StatusOK, map[string]bool{"paused": h.Hub.RecordingPaused()})
}

func (h *Handler) listSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.Hub.TypeSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func (h *Handler) updateSetting(w http.ResponseWriter, r *http.Request) {
	entryType := EntryType(r.PathValue("type"))
	if !ValidEntryType(entryType) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "unknown signal type"})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var input struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.Enabled == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "enabled must be a boolean"})
		return
	}
	deleted, err := h.Hub.SetTypeEnabled(r.Context(), entryType, *input.Enabled)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"type":    entryType,
		"enabled": *input.Enabled,
		"deleted": deleted,
	})
}

func (h *Handler) streamEntries(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	entries, unsubscribe := h.Hub.Subscribe(64)
	defer unsubscribe()
	controls, unsubscribeControls := h.Hub.SubscribeControls(8)
	defer unsubscribeControls()
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case entry, open := <-entries:
			if !open {
				return
			}
			payload, err := json.Marshal(entry)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "id: %s\nevent: entry\ndata: %s\n\n", entry.ID, payload)
			flusher.Flush()
		case control, open := <-controls:
			if !open {
				return
			}
			payload, err := json.Marshal(control)
			if err != nil {
				continue
			}
			_, _ = fmt.Fprintf(w, "event: control\ndata: %s\n\n", payload)
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = w.Write([]byte(": heartbeat\n\n"))
			flusher.Flush()
		}
	}
}

func (h *Handler) listEntries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	ctx := WithoutTrace(r.Context())
	result, err := h.Hub.store.List(ctx, ListFilter{
		Type:      EntryType(q.Get("type")),
		RequestID: q.Get("request_id"),
		Search:    q.Get("search"),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getEntry(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ctx := WithoutTrace(r.Context())
	entry, err := h.Hub.store.Get(ctx, id)
	if err != nil || entry == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "entry not found"})
		return
	}
	batch, err := h.Hub.store.ListByBatch(ctx, entry.BatchID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, buildEntryDetail(entry, batch))
}

func (h *Handler) pruneEntries(w http.ResponseWriter, r *http.Request) {
	n, err := h.Hub.clearAll(WithoutTrace(r.Context()))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, PruneResult{Deleted: n})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// EntrySummary returns a short label for an entry (used in tests).
func EntrySummary(e Entry) string {
	switch e.Type {
	case TypeRequest:
		method, _ := e.Content["method"].(string)
		path, _ := e.Content["path"].(string)
		status, _ := e.Content["status"].(float64)
		return strings.ToUpper(method) + " " + path + " → " + strconv.Itoa(int(status))
	case TypeQuery:
		sql, _ := e.Content["sql"].(string)
		if len(sql) > 60 {
			sql = sql[:60] + "..."
		}
		return sql
	case TypeLog:
		msg, _ := e.Content["message"].(string)
		return msg
	case TypeEvent:
		t, _ := e.Content["event_type"].(string)
		return t
	case TypeNotification:
		kind, _ := e.Content["kind"].(string)
		return kind
	case TypeException:
		msg, _ := e.Content["message"].(string)
		return "panic: " + fmt.Sprint(msg)
	case TypeCache:
		return fmt.Sprint(e.Content["operation"]) + " " + fmt.Sprint(e.Content["key"])
	case TypeRedis:
		return fmt.Sprint(e.Content["command"])
	case TypeJob:
		return fmt.Sprint(e.Content["name"])
	case TypeSchedule:
		return fmt.Sprint(e.Content["name"])
	case TypeMail:
		return fmt.Sprint(e.Content["subject"])
	case TypeHTTPClient:
		return fmt.Sprint(e.Content["method"]) + " " + fmt.Sprint(e.Content["url"])
	case TypeWebSocket:
		return fmt.Sprint(e.Content["event"])
	case TypePerformance:
		return fmt.Sprint(e.Content["name"])
	case TypeMetric:
		return fmt.Sprint(e.Content["name"])
	case TypeCustom:
		return fmt.Sprint(e.Content["name"])
	case TypeTopic:
		return fmt.Sprint(e.Content["action"]) + " " + fmt.Sprint(e.Content["topic"])
	default:
		return string(e.Type)
	}
}
