package microscope

import (
	"context"
	"strings"
)

type ctxKey string

const keyBatchID ctxKey = "microscope_batch_id"
const keySkipTrace ctxKey = "microscope_skip_trace"

// WithBatchID attaches the current request batch to the context.
func WithBatchID(ctx context.Context, batchID string) context.Context {
	return context.WithValue(ctx, keyBatchID, batchID)
}

// BatchIDFromContext returns the batch ID for the current request, if any.
func BatchIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(keyBatchID).(string)
	return v
}

// WithoutTrace marks ctx so microscope query tracing is skipped (internal DB ops).
func WithoutTrace(ctx context.Context) context.Context {
	return context.WithValue(ctx, keySkipTrace, true)
}

func traceSkipped(ctx context.Context) bool {
	v, _ := ctx.Value(keySkipTrace).(bool)
	return v
}

// IsMicroscopePath reports whether path belongs to the microscope dashboard/API.
func IsMicroscopePath(path, prefix string) bool {
	if prefix == "" {
		prefix = "/microscope"
	}
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}
