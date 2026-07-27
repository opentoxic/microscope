package microscope

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

type responseCapture struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *responseCapture) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseCapture) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.body.Len() < 65536 {
		_, _ = w.body.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// Middleware records HTTP requests into microscope.
func (h *Hub) Middleware() func(http.Handler) http.Handler {
	prefix := h.cfg.pathPrefix()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, prefix) {
				next.ServeHTTP(w, r)
				return
			}

			batchID := newID()
			rc := RequestMetaFrom(r.Context())

			ctx := WithBatchID(r.Context(), batchID)
			r = r.WithContext(ctx)

			var reqBody []byte
			if r.Body != nil && r.ContentLength != 0 {
				reqBody, _ = io.ReadAll(io.LimitReader(r.Body, int64(h.cfg.MaxBodyBytes)))
				r.Body = io.NopCloser(bytes.NewReader(reqBody))
			}

			start := time.Now()
			capture := &responseCapture{ResponseWriter: w}
			next.ServeHTTP(capture, r)
			duration := time.Since(start)

			status := capture.status
			if status == 0 {
				status = http.StatusOK
			}

			headers := make(map[string][]string, len(r.Header))
			for k, v := range r.Header {
				headers[k] = append([]string(nil), v...)
			}

			h.Record(ctx, Entry{
				ID:            batchID,
				BatchID:       batchID,
				Type:          TypeRequest,
				RequestID:     rc.RequestID,
				CorrelationID: rc.CorrelationID,
				Tags:          []string{"method:" + r.Method, "status:" + http.StatusText(status)},
				Content: map[string]any{
					"method":        r.Method,
					"path":          r.URL.Path,
					"query":         r.URL.RawQuery,
					"status":        status,
					"duration_ms":   duration.Milliseconds(),
					"ip":            rc.IPAddress,
					"user_agent":    rc.UserAgent,
					"headers":       h.SanitizeHeaders(headers),
					"request_body":  h.SanitizeJSON(reqBody),
					"response_body": h.SanitizeJSON(capture.body.Bytes()),
				},
				CreatedAt: time.Now().UTC(),
			})
		})
	}
}

// RecoverMiddleware records panics as exception entries.
func (h *Hub) RecoverMiddleware(log interface {
	ErrorContext(context.Context, string, ...any)
}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					rc := RequestMetaFrom(r.Context())
					h.Record(r.Context(), Entry{
						Type:          TypeException,
						RequestID:     rc.RequestID,
						CorrelationID: rc.CorrelationID,
						Tags:          []string{"panic"},
						Content: map[string]any{
							"message": rec,
							"path":    r.URL.Path,
							"method":  r.Method,
							"stack":   string(debug.Stack()),
						},
						CreatedAt: time.Now().UTC(),
					})
					log.ErrorContext(r.Context(), "panic recovered", "error", rec)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
