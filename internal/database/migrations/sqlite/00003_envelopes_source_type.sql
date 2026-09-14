-- +goose Up
-- Every envelope is created with a source and type supplied by the service
-- layer, so the DEFAULTs only exist to satisfy SQLite's requirement when
-- adding NOT NULL columns; they are never written by the application.
ALTER TABLE envelopes ADD COLUMN source_id TEXT NOT NULL DEFAULT '';
ALTER TABLE envelopes ADD COLUMN source TEXT NOT NULL DEFAULT '';
ALTER TABLE envelopes ADD COLUMN type TEXT NOT NULL DEFAULT '';

ALTER TABLE envelopes DROP COLUMN routing_key;

CREATE INDEX idx_envelopes_source_id ON envelopes (source_id);

-- +goose Down
DROP INDEX idx_envelopes_source_id;

ALTER TABLE envelopes ADD COLUMN routing_key TEXT NOT NULL DEFAULT '';

ALTER TABLE envelopes DROP COLUMN type;
ALTER TABLE envelopes DROP COLUMN source;
ALTER TABLE envelopes DROP COLUMN source_id;
