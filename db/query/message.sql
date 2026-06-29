-- name: CreateMessage :one
INSERT INTO messages (
    topic_name,
    payload
) VALUES (?, ?)
RETURNING *;

-- name: GetMessage :one
SELECT *
FROM messages
WHERE offset = ?
LIMIT 1;

-- name: PullMessages :many
SELECT *
FROM messages
WHERE topic_name = ?
  AND offset > ?
ORDER BY offset ASC
LIMIT ?;

-- name: ListMessagesByTopic :many
SELECT *
FROM messages
WHERE topic_name = ?
ORDER BY offset ASC;

-- name: DeleteMessagesBefore :exec
DELETE FROM messages
WHERE created_at < ?;

-- name: DeleteAllMessages :exec
DELETE FROM messages;