package migrate

import (
	"context"

	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opentoxic/microscope/adaptor/go"
)

// Up applies embedded microscope migrations idempotently.
func Up(ctx context.Context, pool *pgxpool.Pool) error {
	return microscope.MigrateUp(ctx, pool)
}

// Source returns a golang-migrate source driver for embedded migrations.
func Source() (source.Driver, error) {
	return iofs.New(microscope.MigrationFS(), "migrations")
}
