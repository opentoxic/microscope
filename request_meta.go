package microscope

import (
	"context"
	"net/http"
)

type metaKey struct{}

// RequestMeta carries request-scoped identifiers for microscope entries.
type RequestMeta struct {
	RequestID     string
	CorrelationID string
	IPAddress     string
	UserAgent     string
}

// WithRequestMeta attaches request metadata to ctx.
func WithRequestMeta(ctx context.Context, meta RequestMeta) context.Context {
	return context.WithValue(ctx, metaKey{}, meta)
}

// RequestMetaFrom returns request metadata from ctx.
func RequestMetaFrom(ctx context.Context) RequestMeta {
	meta, _ := ctx.Value(metaKey{}).(RequestMeta)
	return meta
}

// BridgeMiddleware copies request metadata from a host-provided extractor into microscope context.
func BridgeMiddleware(extract func(context.Context) RequestMeta) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := WithRequestMeta(r.Context(), extract(r.Context()))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
