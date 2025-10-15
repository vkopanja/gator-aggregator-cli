-- +goose Up
ALTER TABLE feeds
    ADD COLUMN IF NOT EXISTS last_fetched_at TIMESTAMP;

CREATE INDEX idx_feed_last_fetched_at ON feeds (last_fetched_at);

-- +goose Down
ALTER TABLE feeds
    DROP COLUMN IF EXISTS last_fetched_at;
