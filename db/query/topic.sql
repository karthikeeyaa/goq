-- name: CreateTopic :one
INSERT INTO topics (
    name,
    retention_seconds,
    archive_file,
    mode
) VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetTopic :one
SELECT *
FROM topics
WHERE name = ?
LIMIT 1;

-- name: ListTopics :many
SELECT *
FROM topics
ORDER BY name ASC;

-- name: DeleteTopic :exec
DELETE FROM topics
WHERE name = ?;

-- name: DeleteAllTopics :exec
DELETE FROM topics;