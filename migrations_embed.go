package microscope

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*
var migrationFS embed.FS

// MigrationFS returns embedded SQL migrations for golang-migrate.
// Use with github.com/golang-migrate/migrate/v4/source/iofs:
//
//	source, err := iofs.New(microscope.MigrationFS(), "migrations")
func MigrationFS() fs.FS {
	return migrationFS
}

// MigrationFiles lists up migrations in apply order.
func MigrationFiles() []string {
	return []string{
		"001_microscope.up.sql",
		"002_microscope_settings.up.sql",
		"003_microscope_options.up.sql",
	}
}
