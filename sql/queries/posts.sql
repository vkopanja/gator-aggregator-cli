-- name: CreatePost :one
INSERT INTO posts (title, url, description, published_at, feed_id, created_at, updated_at)
VALUES ($1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7)
RETURNING *;

-- name: GetPostsForUser :many
SELECT p.*
FROM posts p
         JOIN feeds f ON f.id = p.feed_id
         JOIN feed_follows ff ON ff.feed_id = f.id
WHERE ff.user_id = $1
ORDER BY p.published_at DESC NULLS LAST, p.created_at DESC
LIMIT $2;
