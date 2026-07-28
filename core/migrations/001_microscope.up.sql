-- Migration: 001_microscope
-- Dev-only observability entries.

CREATE TABLE IF NOT EXISTS microscope_entries (
    id              TEXT        PRIMARY KEY,
    batch_id        TEXT        NOT NULL,
    type            TEXT        NOT NULL,
    request_id      TEXT,
    correlation_id  TEXT,
    tags            JSONB       NOT NULL DEFAULT '[]',
    content         JSONB       NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS microscope_entries_batch_id_idx ON microscope_entries (batch_id);
CREATE INDEX IF NOT EXISTS microscope_entries_type_created_idx ON microscope_entries (type, created_at DESC);
CREATE INDEX IF NOT EXISTS microscope_entries_request_id_idx ON microscope_entries (request_id);
