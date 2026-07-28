package microscope

import (
	"context"
	"time"
)

// Enabled reports whether microscope should be active for the given app environment.
func Enabled(appEnv string, cfg Config) bool {
	if !cfg.Enabled {
		return false
	}
	allowed := cfg.AllowedEnvs
	if len(allowed) == 0 {
		allowed = DefaultAllowedEnvs()
	}
	for _, env := range allowed {
		if env == appEnv {
			return true
		}
	}
	return false
}

// PruneResult is returned by manual prune API.
type PruneResult struct {
	Deleted int64 `json:"deleted"`
}

// prune removes entries older than the configured retention period (background job).
func (h *Hub) prune(ctx context.Context) (int64, error) {
	cutoff := time.Now().UTC().Add(-h.cfg.retention())
	return h.store.Prune(ctx, cutoff)
}

// clearAll removes every microscope entry (manual dashboard action).
func (h *Hub) clearAll(ctx context.Context) (int64, error) {
	deleted, err := h.store.ClearAll(ctx)
	if err == nil {
		h.publishControl(ControlEvent{Action: "clear-all", Deleted: deleted})
	}
	return deleted, err
}
