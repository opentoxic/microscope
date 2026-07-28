package microscope

import (
	"encoding/json"
	"strings"
)

var sensitiveKeys = map[string]struct{}{
	"password":         {},
	"password_hash":    {},
	"new_password":     {},
	"old_password":     {},
	"current_password": {},
	"refresh_token":    {},
	"access_token":     {},
	"token":            {},
	"otp":              {},
	"code":             {},
	"secret":           {},
	"encryption_key":   {},
	"authorization":    {},
	"mfa_secret":       {},
	"backup_codes":     {},
}

var sensitiveHeaderKeys = map[string]struct{}{
	"authorization": {},
	"cookie":        {},
	"x-api-key":     {},
	"x-auth-token":  {},
}

// RedactMap returns a copy of m with sensitive values masked.
func RedactMap(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = redactValue(k, v)
	}
	return out
}

// RedactHeaders returns a copy of headers with sensitive values masked.
func RedactHeaders(headers map[string][]string) map[string][]string {
	if headers == nil {
		return nil
	}
	out := make(map[string][]string, len(headers))
	for k, vals := range headers {
		if _, ok := sensitiveHeaderKeys[strings.ToLower(k)]; ok {
			out[k] = []string{"[REDACTED]"}
			continue
		}
		out[k] = append([]string(nil), vals...)
	}
	return out
}

func redactValue(key string, v any) any {
	if _, ok := sensitiveKeys[strings.ToLower(key)]; ok {
		return "[REDACTED]"
	}
	switch val := v.(type) {
	case map[string]any:
		return RedactMap(val)
	case []any:
		out := make([]any, len(val))
		for i, item := range val {
			out[i] = redactValue(key, item)
		}
		return out
	default:
		return v
	}
}

// RedactJSON parses JSON bytes, redacts sensitive fields, and re-marshals.
func RedactJSON(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return string(body)
	}
	redacted := redactValue("", data)
	out, err := json.Marshal(redacted)
	if err != nil {
		return string(body)
	}
	return string(out)
}
