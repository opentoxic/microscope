package microscope

import (
	"context"
	"log/slog"
	"strings"
	"time"
)

// SlogHandler tees log records to microscope.
type SlogHandler struct {
	inner  slog.Handler
	hub    *Hub
	prefix string
}

// NewSlogHandler wraps h and records log entries to microscope.
func NewSlogHandler(h slog.Handler, hub *Hub) *SlogHandler {
	return &SlogHandler{inner: h, hub: hub, prefix: hub.cfg.pathPrefix()}
}

func (h *SlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *SlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if h.skipStdout(r) {
		return nil
	}
	if h.hub != nil && !h.skipRecording(r) {
		rc := RequestMetaFrom(ctx)
		attrs := map[string]any{
			"level":   r.Level.String(),
			"message": r.Message,
		}
		r.Attrs(func(a slog.Attr) bool {
			attrs[a.Key] = a.Value.Any()
			return true
		})
		h.hub.Record(ctx, Entry{
			Type:          TypeLog,
			RequestID:     rc.RequestID,
			CorrelationID: rc.CorrelationID,
			Tags:          []string{"level:" + r.Level.String()},
			Content:       RedactMap(attrs),
			CreatedAt:     time.Now().UTC(),
		})
	}
	return h.inner.Handle(ctx, r)
}

func (h *SlogHandler) skipRecording(r slog.Record) bool {
	// The request watcher already owns access-log data. Recording the "request"
	// slog record as well creates a visually duplicated batch without adding context.
	return r.Message == "request" || h.skipMicroscope(r)
}

func (h *SlogHandler) skipStdout(r slog.Record) bool {
	return h.skipMicroscope(r)
}

func (h *SlogHandler) skipMicroscope(r slog.Record) bool {
	if strings.HasPrefix(r.Message, "microscope:") {
		return true
	}
	if r.Message != "request" {
		return false
	}
	var path string
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "path" {
			path, _ = a.Value.Any().(string)
			return false
		}
		return true
	})
	return IsMicroscopePath(path, h.prefix)
}

func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SlogHandler{inner: h.inner.WithAttrs(attrs), hub: h.hub, prefix: h.prefix}
}

func (h *SlogHandler) WithGroup(name string) slog.Handler {
	return &SlogHandler{inner: h.inner.WithGroup(name), hub: h.hub, prefix: h.prefix}
}
