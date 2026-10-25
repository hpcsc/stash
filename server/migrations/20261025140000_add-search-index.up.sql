CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX IF NOT EXISTS idx_items_body_trgm ON items USING GIN (body gin_trgm_ops);