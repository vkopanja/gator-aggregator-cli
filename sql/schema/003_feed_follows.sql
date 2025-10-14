-- +goose Up
CREATE TABLE feed_follows
(
    id         UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
    user_id    UUID             NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    feed_id    UUID             NOT NULL REFERENCES feeds (id) ON DELETE CASCADE,
    created_at TIMESTAMP        NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP        NOT NULL
);

CREATE UNIQUE INDEX idx_feed_follows_user_feed_uq ON feed_follows (user_id, feed_id);

-- +goose Down
DROP TABLE feed_follows;