-- name: CreateMessage :one
INSERT INTO messages (
    id,
    topic_name,
    operation_name,
    payload,
    status
) VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMessage :one
SELECT * FROM messages
WHERE id = ? LIMIT 1;

-- name: UpdateMessageStatus :one
UPDATE messages
SET status = ?
WHERE id = ?
RETURNING *;

-- name: ListMessagesByStatus :many
SELECT * FROM messages
WHERE status = ?
ORDER BY created_at DESC;

-- name: ListMessages :many
SELECT * FROM messages
ORDER BY created_at DESC;

-- name: DeleteMessage :exec
DELETE FROM messages
WHERE id = ?;
