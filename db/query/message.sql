-- name: CreateMessage :one
INSERT INTO messages (
    topic_name,
    payload
) VALUES (?, ?)
RETURNING *;

-- name: PullMessages :many
SELECT *
FROM messages
WHERE topic_name = ?
  AND offset > ?
ORDER BY offset ASC
LIMIT ?;

-- name: GetHeadOffset :one
SELECT COALESCE(MAX(offset), 0) AS head_offset
FROM messages
WHERE topic_name = ?;

-- name: GetEarliestOffset :one
SELECT COALESCE(MIN(offset), 0) AS earliest_offset
FROM messages
WHERE topic_name = ?;