package microscope

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func renameLegacyTelescopeTables(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		DO $rename$
		BEGIN
			IF to_regclass('public.telescope_entries') IS NOT NULL
				AND to_regclass('public.microscope_entries') IS NULL THEN
				ALTER TABLE telescope_entries RENAME TO microscope_entries;
			END IF;
			IF to_regclass('public.telescope_settings') IS NOT NULL
				AND to_regclass('public.microscope_settings') IS NULL THEN
				ALTER TABLE telescope_settings RENAME TO microscope_settings;
			END IF;
			IF to_regclass('public.telescope_entries_batch_id_idx') IS NOT NULL
				AND to_regclass('public.microscope_entries_batch_id_idx') IS NULL THEN
				ALTER INDEX telescope_entries_batch_id_idx RENAME TO microscope_entries_batch_id_idx;
			END IF;
			IF to_regclass('public.telescope_entries_type_created_idx') IS NOT NULL
				AND to_regclass('public.microscope_entries_type_created_idx') IS NULL THEN
				ALTER INDEX telescope_entries_type_created_idx RENAME TO microscope_entries_type_created_idx;
			END IF;
			IF to_regclass('public.telescope_entries_request_id_idx') IS NOT NULL
				AND to_regclass('public.microscope_entries_request_id_idx') IS NULL THEN
				ALTER INDEX telescope_entries_request_id_idx RENAME TO microscope_entries_request_id_idx;
			END IF;
		END
		$rename$`)
	if err != nil {
		return fmt.Errorf("rename legacy telescope tables: %w", err)
	}
	return nil
}
