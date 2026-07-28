package microscope

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// HTTPOptions configures HTTP route registration and middleware for an Integration.
type HTTPOptions struct {
	// Bridge copies host request metadata into microscope context. Optional.
	// When nil, RequestIDFromContext is used.
	Bridge func(context.Context) RequestMeta
	// RequestID runs before bridge. Defaults to RequestIDMiddleware when nil.
	RequestID func(http.Handler) http.Handler
	// AccessLog is the host application's request logger middleware. Optional.
	AccessLog func(http.Handler) http.Handler
	// NoAccessLogSkip disables skipping AccessLog on microscope paths (skip is default when AccessLog is set).
	NoAccessLogSkip bool
	// Extra middleware runs after recover and before request ID (e.g. CORS).
	Extra []func(http.Handler) http.Handler
}

// RegisterRoutes mounts the microscope API and embedded UI on mux.
func (i *Integration) RegisterRoutes(mux *http.ServeMux) {
	if i.hub == nil || mux == nil {
		return
	}
	(&Handler{Hub: i.hub}).RegisterRoutes(mux)
}

// HTTPMiddlewares returns middleware to apply after host request-ID middleware.
// Order: bridge (optional), hub recording, access-log skip (optional).
func (i *Integration) HTTPMiddlewares(opts HTTPOptions) []func(http.Handler) http.Handler {
	if i.hub == nil {
		return nil
	}
	bridge := opts.Bridge
	if bridge == nil {
		bridge = func(ctx context.Context) RequestMeta {
			return RequestMeta{RequestID: RequestIDFromContext(ctx)}
		}
	}
	var middlewares []func(http.Handler) http.Handler
	middlewares = append(middlewares, BridgeMiddleware(bridge))
	middlewares = append(middlewares, i.hub.Middleware())
	if opts.AccessLog != nil {
		if opts.NoAccessLogSkip {
			middlewares = append(middlewares, opts.AccessLog)
		} else {
			middlewares = append(middlewares, skipPrefixMiddleware(i.cfg.pathPrefix(), opts.AccessLog))
		}
	}
	return middlewares
}

// HTTPHandler returns a fully wrapped handler: recover → extra → request ID → bridge → record → access log.
func (i *Integration) HTTPHandler(mux http.Handler, opts HTTPOptions) http.Handler {
	log := slog.Default()
	if i != nil && i.hub != nil && i.hub.log != nil {
		log = i.hub.log
	}

	handler := mux
	if i != nil {
		for _, mw := range i.HTTPMiddlewares(opts) {
			handler = mw(handler)
		}
	}

	requestID := opts.RequestID
	if requestID == nil {
		requestID = RequestIDMiddleware()
	}
	handler = requestID(handler)

	for j := len(opts.Extra) - 1; j >= 0; j-- {
		handler = opts.Extra[j](handler)
	}

	if i == nil {
		return defaultRecover(log)(handler)
	}
	return i.RecoverMiddleware(log)(handler)
}

// RecoverMiddleware returns panic recovery that records exceptions in microscope.
func (i *Integration) RecoverMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	if i.hub == nil {
		return defaultRecover(log)
	}
	return i.hub.RecoverMiddleware(log)
}

func skipPrefixMiddleware(prefix string, inner func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	base := inner
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if prefix != "" && (r.URL.Path == prefix || strings.HasPrefix(r.URL.Path, prefix+"/")) {
				next.ServeHTTP(w, r)
				return
			}
			base(next).ServeHTTP(w, r)
		})
	}
}

func defaultRecover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					if log != nil {
						log.Error("panic recovered", "error", rec)
					}
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// SimpleAccessLog returns a basic slog request logger for use with HTTPOptions.AccessLog.
func SimpleAccessLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			next.ServeHTTP(w, r)
			if log != nil {
				log.Info("request",
					"method", r.Method,
					"path", r.URL.Path,
					"duration", time.Since(started),
				)
			}
		})
	}
}
