-- +goose Up
ALTER TABLE sources ADD COLUMN is_active INTEGER NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE sources DROP COLUMN is_active;
