-- +goose Up
-- Every destination is created with a type supplied by the service layer, so
-- the DEFAULT only exists to satisfy SQLite's requirement when adding a NOT
-- NULL column; it is never relied upon by the application. "webhook" is
-- currently the only supported type.
ALTER TABLE destinations ADD COLUMN type TEXT NOT NULL DEFAULT 'webhook';

-- +goose Down
ALTER TABLE destinations DROP COLUMN type;
