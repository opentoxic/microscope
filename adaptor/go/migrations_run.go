package microscope

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schemaMigrationsTable = "microscope_schema_migrations"

// MigrateUp applies embedded microscope migrations idempotently.
func MigrateUp(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("microscope: pool is required")
	}
	if err := renameLegacyTelescopeTables(ctx, pool); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`, schemaMigrationsTable)); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	for _, name := range MigrationFiles() {
		version := strings.TrimSuffix(name, ".up.sql")
		var exists bool
		if err := pool.QueryRow(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE version = $1)", schemaMigrationsTable), version).Scan(&exists); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if exists {
			continue
		}

		data, err := readEmbeddedMigration(name)
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if _, err := pool.Exec(ctx, fmt.Sprintf("INSERT INTO %s (version) VALUES ($1)", schemaMigrationsTable), version); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
	}
	return nil
}

func readEmbeddedMigration(name string) ([]byte, error) {
	f, err := MigrationFS().Open("migrations/" + name)
	if err != nil {
		return nil, fmt.Errorf("open migration %s: %w", name, err)
	}
	defer f.Close()
	return io.ReadAll(f)
}
