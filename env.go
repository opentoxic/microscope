package microscope

import (
	"os"
	"strconv"
	"strings"
)

// ConfigFromEnv reads microscope settings from environment variables.
// Unset variables use DefaultConfig() values. MICROSCOPE_ENABLED defaults to true when unset.
func ConfigFromEnv() Config {
	cfg := DefaultConfig()
	if v := os.Getenv("MICROSCOPE_ENABLED"); v != "" {
		cfg.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("MICROSCOPE_PATH"); v != "" {
		cfg.Path = v
	}
	if v := os.Getenv("MICROSCOPE_RETENTION_HOURS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RetentionHours = n
		}
	}
	if v := os.Getenv("MICROSCOPE_MAX_BODY_BYTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.MaxBodyBytes = n
		}
	}
	if v := os.Getenv("MICROSCOPE_ALLOWED_ENVS"); v != "" {
		cfg.AllowedEnvs = splitEnvList(v)
	}
	if v := os.Getenv("MICROSCOPE_AUTO_MIGRATE"); v != "" {
		cfg.AutoMigrate = v == "true" || v == "1"
	}
	if v := os.Getenv("MICROSCOPE_REDACT_SENSITIVE"); v != "" {
		cfg.RedactSensitive = v == "true" || v == "1"
	}
	return cfg
}

// MergeConfig overlays overrides onto base. Non-zero override fields replace base values.
func MergeConfig(base, overrides Config) Config {
	out := base
	out.Enabled = overrides.Enabled
	if overrides.Path != "" {
		out.Path = overrides.Path
	}
	if overrides.RetentionHours > 0 {
		out.RetentionHours = overrides.RetentionHours
	}
	if overrides.MaxBodyBytes > 0 {
		out.MaxBodyBytes = overrides.MaxBodyBytes
	}
	if len(overrides.AllowedEnvs) > 0 {
		out.AllowedEnvs = append([]string(nil), overrides.AllowedEnvs...)
	}
	out.AutoMigrate = overrides.AutoMigrate
	out.RedactSensitive = overrides.RedactSensitive
	return out
}

func splitEnvList(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOr(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
