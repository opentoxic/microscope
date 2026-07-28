CREATE TABLE IF NOT EXISTS microscope_options (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO microscope_options (key, value)
VALUES ('redact_sensitive', 'false'::jsonb)
ON CONFLICT DO NOTHING;
