package microscope

import (
	"context"
	"net/http"
)

type requestIDKey struct{}

// RequestIDFromContext returns the request ID set by RequestIDMiddleware.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// RequestIDMiddleware assigns X-Request-ID and stores it in context and RequestMeta.
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				id = newID()
			}
			w.Header().Set("X-Request-ID", id)
			ctx := context.WithValue(r.Context(), requestIDKey{}, id)
			meta := RequestMetaFrom(ctx)
			meta.RequestID = id
			ctx = WithRequestMeta(ctx, meta)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
