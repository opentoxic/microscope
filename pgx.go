package microscope

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type queryTraceData struct {
	start time.Time
	sql   string
	args  []any
}

// QueryTracer implements pgx.QueryTracer and records SQL to a Hub once bound.
type QueryTracer struct {
	hub *Hub
}

// NewQueryTracer creates a tracer that can be bound to a Hub after pool creation.
func NewQueryTracer() *QueryTracer {
	return &QueryTracer{}
}

func (t *QueryTracer) bind(h *Hub) {
	t.hub = h
}

// Bind attaches the hub used for recording queries.
func (t *QueryTracer) Bind(h *Hub) {
	t.hub = h
}

// TraceQueryStart implements pgx.QueryTracer.
func (t *QueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if traceSkipped(ctx) || isMicroscopeSQL(data.SQL) {
		return ctx
	}
	return context.WithValue(ctx, queryTraceKey{}, queryTraceData{
		start: time.Now(),
		sql:   data.SQL,
		args:  data.Args,
	})
}

// TraceQueryEnd implements pgx.QueryTracer.
func (t *QueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	if traceSkipped(ctx) || t.hub == nil {
		return
	}
	trace, ok := ctx.Value(queryTraceKey{}).(queryTraceData)
	if !ok {
		return
	}

	rc := RequestMetaFrom(ctx)
	duration := time.Since(trace.start)

	content := map[string]any{
		"sql":         trace.sql,
		"args":        redactArgs(trace.args),
		"duration_ms": duration.Milliseconds(),
	}
	if data.Err != nil {
		content["error"] = data.Err.Error()
	}
	if data.CommandTag.String() != "" {
		content["command_tag"] = data.CommandTag.String()
	}

	t.hub.Record(ctx, Entry{
		Type:          TypeQuery,
		RequestID:     rc.RequestID,
		CorrelationID: rc.CorrelationID,
		Tags:          []string{"sql"},
		Content:       content,
		CreatedAt:     time.Now().UTC(),
	})
}

type queryTraceKey struct{}

func isMicroscopeSQL(sql string) bool {
	return strings.Contains(strings.ToLower(sql), "microscope_entries")
}

func redactArgs(args []any) []any {
	if len(args) == 0 {
		return nil
	}
	out := make([]any, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case map[string]any:
			out[i] = RedactMap(v)
		case []byte:
			out[i] = "[bytes]"
		default:
			out[i] = v
		}
	}
	return out
}
