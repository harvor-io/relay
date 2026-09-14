-- +goose Up
CREATE TABLE source_keys (
    id         TEXT PRIMARY KEY,
    source_id  TEXT NOT NULL,
    secret_id  TEXT NOT NULL,
    name       TEXT NOT NULL,
    is_active  INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_source_keys_source_id ON source_keys (source_id);
CREATE UNIQUE INDEX idx_source_keys_secret_id ON source_keys (secret_id);

-- +goose Down
DROP INDEX idx_source_keys_secret_id;
DROP INDEX idx_source_keys_source_id;
DROP TABLE source_keys;
