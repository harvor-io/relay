-- +goose Up
-- Every source created through the API supplies a slug, so the DEFAULT only
-- exists to satisfy SQLite's requirement when adding a NOT NULL column; it is
-- never written by the application.
ALTER TABLE sources ADD COLUMN slug TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX idx_sources_slug ON sources (slug);

-- +goose Down
DROP INDEX idx_sources_slug;
ALTER TABLE sources DROP COLUMN slug;
