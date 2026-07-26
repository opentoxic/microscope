package microscope

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store persists microscope entries.
type Store interface {
	Insert(ctx context.Context, e Entry) error
	Get(ctx context.Context, id string) (*Entry, error)
	List(ctx context.Context, f ListFilter) (ListResult, error)
	ListByBatch(ctx context.Context, batchID string) ([]Entry, error)
	Prune(ctx context.Context, olderThan time.Time) (int64, error)
	ClearAll(ctx context.Context) (int64, error)
	ListTypeSettings(ctx context.Context) ([]TypeSetting, error)
	SetTypeEnabled(ctx context.Context, entryType EntryType, enabled bool) (int64, error)
}

// TypeSetting controls both ingestion and retention for one signal type.
type TypeSetting struct {
	Type    EntryType `json:"type"`
	Enabled bool      `json:"enabled"`
	Count   int64     `json:"count"`
}

// PostgresStore stores entries in PostgreSQL.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a store backed by the given pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

func (s *PostgresStore) Insert(ctx context.Context, e Entry) error {
	tags, err := encodeTags(e.Tags)
	if err != nil {
		return err
	}
	content, err := encodeContent(e.Content)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO microscope_entries (id, batch_id, type, request_id, correlation_id, tags, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		e.ID, e.BatchID, string(e.Type), nullIfEmpty(e.RequestID), nullIfEmpty(e.CorrelationID), tags, content, e.CreatedAt,
	)
	return err
}

func (s *PostgresStore) Get(ctx context.Context, id string) (*Entry, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, batch_id, type, request_id, correlation_id, tags, content, created_at
		FROM microscope_entries WHERE id = $1`, id)
	return scanEntry(row)
}

func (s *PostgresStore) List(ctx context.Context, f ListFilter) (ListResult, error) {
	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	where := "WHERE 1=1"
	args := []any{}
	argN := 1

	if f.Type != "" {
		where += fmt.Sprintf(" AND type = $%d", argN)
		args = append(args, string(f.Type))
		argN++
	}
	if f.RequestID != "" {
		where += fmt.Sprintf(" AND request_id = $%d", argN)
		args = append(args, f.RequestID)
		argN++
	}
	if f.Search != "" {
		where += fmt.Sprintf(" AND (content::text ILIKE $%d OR request_id ILIKE $%d)", argN, argN)
		args = append(args, "%"+f.Search+"%")
		argN++
	}

	countSQL := "SELECT COUNT(*) FROM microscope_entries " + where
	var total int
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return ListResult{}, err
	}

	listSQL := fmt.Sprintf(`
		SELECT id, batch_id, type, request_id, correlation_id, tags, content, created_at
		FROM microscope_entries %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, argN, argN+1)
	args = append(args, limit, f.Offset)

	rows, err := s.pool.Query(ctx, listSQL, args...)
	if err != nil {
		return ListResult{}, err
	}
	defer rows.Close()

	entries := make([]Entry, 0, limit)
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return ListResult{}, err
		}
		entries = append(entries, *e)
	}
	return ListResult{Entries: entries, Total: total}, rows.Err()
}

func (s *PostgresStore) ListByBatch(ctx context.Context, batchID string) ([]Entry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, batch_id, type, request_id, correlation_id, tags, content, created_at
		FROM microscope_entries
		WHERE batch_id = $1
		ORDER BY created_at ASC`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, *e)
	}
	return entries, rows.Err()
}

func (s *PostgresStore) Prune(ctx context.Context, olderThan time.Time) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM microscope_entries WHERE created_at < $1`, olderThan)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *PostgresStore) ClearAll(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM microscope_entries`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *PostgresStore) ListTypeSettings(ctx context.Context) ([]TypeSetting, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT configured.type,
		       COALESCE(settings.enabled, TRUE),
		       COALESCE(entries.count, 0)
		FROM unnest($1::text[]) AS configured(type)
		LEFT JOIN microscope_settings settings ON settings.type = configured.type
		LEFT JOIN (
			SELECT type, COUNT(*) AS count
			FROM microscope_entries
			GROUP BY type
		) entries ON entries.type = configured.type
		ORDER BY array_position($1::text[], configured.type)`,
		entryTypeStrings(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make([]TypeSetting, 0, len(AllEntryTypes))
	for rows.Next() {
		var setting TypeSetting
		var entryType string
		if err := rows.Scan(&entryType, &setting.Enabled, &setting.Count); err != nil {
			return nil, err
		}
		setting.Type = EntryType(entryType)
		settings = append(settings, setting)
	}
	return settings, rows.Err()
}

func (s *PostgresStore) SetTypeEnabled(ctx context.Context, entryType EntryType, enabled bool) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx, `
		INSERT INTO microscope_settings (type, enabled, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (type) DO UPDATE
		SET enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at`,
		string(entryType), enabled,
	); err != nil {
		return 0, err
	}

	var deleted int64
	if !enabled {
		tag, deleteErr := tx.Exec(ctx, `DELETE FROM microscope_entries WHERE type = $1`, string(entryType))
		if deleteErr != nil {
			return 0, deleteErr
		}
		deleted = tag.RowsAffected()
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return deleted, nil
}

func entryTypeStrings() []string {
	types := make([]string, 0, len(AllEntryTypes))
	for _, entryType := range AllEntryTypes {
		types = append(types, string(entryType))
	}
	return types
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanEntry(row rowScanner) (*Entry, error) {
	var e Entry
	var entryType string
	var requestID, correlationID *string
	var tagsJSON, contentJSON []byte
	if err := row.Scan(&e.ID, &e.BatchID, &entryType, &requestID, &correlationID, &tagsJSON, &contentJSON, &e.CreatedAt); err != nil {
		return nil, err
	}
	e.Type = EntryType(entryType)
	if requestID != nil {
		e.RequestID = *requestID
	}
	if correlationID != nil {
		e.CorrelationID = *correlationID
	}
	if err := json.Unmarshal(tagsJSON, &e.Tags); err != nil {
		e.Tags = []string{}
	}
	if err := json.Unmarshal(contentJSON, &e.Content); err != nil {
		e.Content = map[string]any{}
	}
	return &e, nil
}

func nullIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
