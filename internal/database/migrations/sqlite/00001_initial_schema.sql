-- +goose Up
CREATE TABLE sources (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

CREATE TABLE destinations (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);

CREATE TABLE envelopes (
    id          TEXT PRIMARY KEY,
    routing_key TEXT NOT NULL,
    message     TEXT NOT NULL,
    created_at  TEXT NOT NULL
);

CREATE INDEX idx_envelopes_created_at ON envelopes (created_at DESC);

-- +goose Down
DROP INDEX idx_envelopes_created_at;
DROP TABLE envelopes;
DROP TABLE destinations;
DROP TABLE sources;
