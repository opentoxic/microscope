package microscope

import "time"

// Config holds microscope runtime options.
type Config struct {
	Enabled         bool
	Path            string
	RetentionHours  int
	MaxBodyBytes    int
	AllowedEnvs     []string
	AutoMigrate     *bool
	RedactSensitive *bool
}

// BoolPtr returns a pointer to v.
func BoolPtr(v bool) *bool {
	return &v
}

// BoolValue returns *p when p is non-nil, otherwise defaultVal.
func BoolValue(p *bool, defaultVal bool) bool {
	if p == nil {
		return defaultVal
	}
	return *p
}

// DefaultAllowedEnvs lists app environments where microscope may run.
func DefaultAllowedEnvs() []string {
	return []string{"development", "local"}
}

// DefaultConfig returns sensible development defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:        true,
		Path:           "/microscope",
		RetentionHours: 24,
		MaxBodyBytes:   65536,
		AllowedEnvs:    DefaultAllowedEnvs(),
		AutoMigrate:    BoolPtr(true),
	}
}

func (c Config) retention() time.Duration {
	if c.RetentionHours <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(c.RetentionHours) * time.Hour
}

func (c Config) pathPrefix() string {
	if c.Path == "" {
		return "/microscope"
	}
	return c.Path
}
