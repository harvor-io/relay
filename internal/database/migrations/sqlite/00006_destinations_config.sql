-- +goose Up
-- Every destination is created with a config supplied by the service layer,
-- so the DEFAULT only exists to satisfy SQLite's requirement when adding a
-- NOT NULL column; it is never relied upon by the application.
ALTER TABLE destinations ADD COLUMN config TEXT NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE destinations DROP COLUMN config;
