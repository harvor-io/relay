-- +goose Up
-- A subscription is the link between a destination and one topic or pattern,
-- identified by the two of them. It names no source: the topic already starts
-- with the source's slug. Deleting the destination deletes its subscriptions.
-- Each of a subscription's topics becomes a subscription of its own, and rows
-- whose destination is already gone are dropped, since the foreign key would
-- refuse them.
CREATE TABLE subscription_topics (
    destination_id TEXT NOT NULL REFERENCES destinations (id) ON DELETE CASCADE,
    topic          TEXT NOT NULL,
    created_at     TEXT NOT NULL,
    PRIMARY KEY (destination_id, topic)
);

INSERT OR IGNORE INTO subscription_topics (destination_id, topic, created_at)
SELECT s.destination_id, t.value, s.created_at
FROM subscriptions AS s, json_each(s.topics) AS t
WHERE s.destination_id IN (SELECT id FROM destinations);

DROP INDEX idx_subscriptions_destination_source;
DROP TABLE subscriptions;
ALTER TABLE subscription_topics RENAME TO subscriptions;

-- +goose Down
-- Rolling back deletes every subscription: the earlier shape groups topics by
-- source ID, which this one doesn't record.
DROP TABLE subscriptions;

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
