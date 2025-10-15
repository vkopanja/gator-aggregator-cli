-- +goose Up
CREATE TABLE posts
(
    id           UUID      NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    title        TEXT      NOT NULL,
    url          TEXT UNIQUE,
    description  TEXT,
    published_at TIMESTAMP,
    feed_id      UUID      NOT NULL REFERENCES feeds (id),
    created_at   TIMESTAMP NOT NULL             DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE posts;
