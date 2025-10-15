-- name: CreateFeed :one
INSERT INTO feeds (id, name, url, user_id, created_at, updated_at)
VALUES ($1,
        $2,
        $3,
        $4,
        $5,
        $6)
RETURNING *;

-- name: GetFeedsWithUserName :many
SELECT f.*, u.name AS user_name
FROM feeds f
         JOIN users u ON u.id = f.user_id;

-- name: CreateFeedFollow :one
WITH new_feed_follow AS (
    INSERT INTO feed_follows (id, user_id, feed_id, created_at, updated_at)
        VALUES ($1,
                $2,
                $3,
                $4,
                $5)
        RETURNING *)
SELECT nff.*,
       u.name AS user_name,
       f.name AS feed_name
FROM new_feed_follow AS nff
         JOIN feeds f ON f.id = nff.feed_id
         JOIN users u ON u.id = nff.user_id;

-- name: GetFeedByUrl :one
SELECT *
FROM feeds
WHERE url = $1;

-- name: GetFeedsForUser :many
SELECT f.*, u.name AS user_name
FROM feeds f
         JOIN feed_follows ff ON ff.feed_id = f.id
         JOIN users u ON u.id = ff.user_id
WHERE u.name = $1;

-- name: UnfollowFeed :execrows
DELETE
FROM feed_follows
WHERE user_id = $1
  AND feed_id = $2;

-- name: MarkFeedFetched :execrows
UPDATE feeds
SET last_fetched_at = NOW(),
    updated_at      = NOW()
WHERE id = $1;

-- name: GetNextFeedToFetch :one
SELECT *
FROM feeds
ORDER BY last_fetched_at NULLS FIRST
LIMIT 1;