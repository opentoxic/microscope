package microscope

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const redactSensitiveOptionKey = "redact_sensitive"

// RedactSensitive reports whether sensitive fields are masked before storage.
func (h *Hub) RedactSensitive() bool {
	if h == nil {
		return false
	}
	h.redactionMu.RLock()
	defer h.redactionMu.RUnlock()
	return h.redactSensitive
}

// SetRedactSensitive toggles whether sensitive fields are masked before storage.
func (h *Hub) SetRedactSensitive(enabled bool) error {
	if h == nil {
		return nil
	}
	h.redactionMu.Lock()
	if h.redactSensitive == enabled {
		h.redactionMu.Unlock()
		return nil
	}
	h.redactSensitive = enabled
	h.redactionMu.Unlock()

	ctx, cancel := context.WithTimeout(WithoutTrace(context.Background()), 5*time.Second)
	defer cancel()
	raw, err := json.Marshal(enabled)
	if err != nil {
		return err
	}
	if err := h.store.SetOption(ctx, redactSensitiveOptionKey, raw); err != nil {
		return err
	}
	h.publishControl(ControlEvent{Action: "redaction-setting", RedactSensitive: &enabled})
	return nil
}

func (h *Hub) loadRedactionSetting() {
	ctx, cancel := context.WithTimeout(WithoutTrace(context.Background()), 5*time.Second)
	defer cancel()
	raw, err := h.store.GetOption(ctx, redactSensitiveOptionKey)
	if err != nil || raw == nil {
		return
	}
	var enabled bool
	if err := json.Unmarshal(raw, &enabled); err != nil {
		return
	}
	h.redactionMu.Lock()
	h.redactSensitive = enabled
	h.redactionMu.Unlock()
}

// SanitizeMap returns a storage-safe copy of m, optionally masking sensitive values.
func (h *Hub) SanitizeMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	if h != nil && h.RedactSensitive() {
		return RedactMap(m)
	}
	return deepCloneMap(m)
}

// SanitizeHeaders returns a storage-safe copy of headers.
func (h *Hub) SanitizeHeaders(headers map[string][]string) map[string][]string {
	if headers == nil {
		return nil
	}
	if h != nil && h.RedactSensitive() {
		return RedactHeaders(headers)
	}
	out := make(map[string][]string, len(headers))
	for k, vals := range headers {
		out[k] = append([]string(nil), vals...)
	}
	return out
}

// SanitizeJSON returns a storage-safe JSON body string.
func (h *Hub) SanitizeJSON(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	if h != nil && h.RedactSensitive() {
		return RedactJSON(body)
	}
	return string(body)
}

// SanitizeArgs returns a storage-safe copy of SQL arguments.
func (h *Hub) SanitizeArgs(args []any) []any {
	if len(args) == 0 {
		return nil
	}
	if h != nil && h.RedactSensitive() {
		return redactArgs(args)
	}
	out := make([]any, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case map[string]any:
			out[i] = h.SanitizeMap(v)
		case []byte:
			out[i] = string(v)
		default:
			out[i] = v
		}
	}
	return out
}

// SanitizeURL returns a storage-safe request URL.
func (h *Hub) SanitizeURL(request *http.Request) string {
	if request == nil || request.URL == nil {
		return ""
	}
	if h != nil && h.RedactSensitive() {
		return redactedURL(request)
	}
	return request.URL.String()
}

// SanitizeOTP returns a storage-safe OTP value.
func (h *Hub) SanitizeOTP(otp string) string {
	if h != nil && h.RedactSensitive() {
		return "[REDACTED]"
	}
	return otp
}

// TruncateBytes returns s truncated to maxLen bytes.
func TruncateBytes(data []byte, maxLen int) string {
	if maxLen <= 0 || len(data) == 0 {
		return ""
	}
	if len(data) <= maxLen {
		return string(data)
	}
	return string(data[:maxLen])
}

func deepCloneMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = deepCloneValue(v)
	}
	return out
}

func deepCloneValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return deepCloneMap(val)
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = deepCloneValue(item)
		}
		return out
	default:
		return val
	}
}
