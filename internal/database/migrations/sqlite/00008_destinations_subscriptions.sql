-- +goose Up
-- Every destination is created with subscriptions resolved by the service
-- layer (defaulting to an empty array), so the DEFAULT only exists to
-- satisfy SQLite's requirement when adding a NOT NULL column; it is never
-- relied upon by the application.
ALTER TABLE destinations ADD COLUMN subscriptions TEXT NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE destinations DROP COLUMN subscriptions;
