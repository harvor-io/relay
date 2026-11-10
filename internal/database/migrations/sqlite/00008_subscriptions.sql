-- +goose Up
-- topics is a JSON array of the subscription's topics and patterns. source_slug
-- is captured at creation so topics can still be checked against it after the
-- source is deleted.
CREATE TABLE subscriptions (
    id             TEXT PRIMARY KEY,
    destination_id TEXT NOT NULL,
    source_id      TEXT NOT NULL,
    source_slug    TEXT NOT NULL,
    topics         TEXT NOT NULL,
    created_at     TEXT NOT NULL,
    updated_at     TEXT NOT NULL
);

CREATE UNIQUE INDEX idx_subscriptions_destination_source ON subscriptions (destination_id, source_id);

-- +goose Down
DROP INDEX idx_subscriptions_destination_source;
DROP TABLE subscriptions;
