ALTER TABLE items ADD COLUMN IF NOT EXISTS note_blocks JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE TABLE IF NOT EXISTS item_images (
    id             UUID PRIMARY KEY,
    item_id        UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    file_path      TEXT NOT NULL,
    thumbnail_path TEXT,
    file_name      TEXT NOT NULL DEFAULT '',
    file_size      BIGINT NOT NULL DEFAULT 0,
    mime_type      TEXT NOT NULL DEFAULT '',
    sort_order     INTEGER NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS item_images_item_id_sort_order_idx ON item_images (item_id, sort_order);
