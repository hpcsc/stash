CREATE TABLE IF NOT EXISTS items (
    id          UUID PRIMARY KEY,
    type        TEXT NOT NULL,
    body        TEXT NOT NULL,
    source_app  TEXT NOT NULL DEFAULT '',
    source_url  TEXT NOT NULL DEFAULT '',
    captured_at TIMESTAMPTZ NOT NULL,
    version     BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
