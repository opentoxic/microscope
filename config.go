package microscope

import "time"

// Config holds microscope runtime options.
type Config struct {
	Enabled        bool
	Path           string
	RetentionHours int
	MaxBodyBytes   int
	AllowedEnvs    []string
	AutoMigrate    bool
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
		AutoMigrate:    true,
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
