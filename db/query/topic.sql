-- name: GetTopic :one
SELECT * FROM topics
WHERE id = ? LIMIT 1;

-- name: GetTopicByName :one
SELECT * FROM topics
WHERE name = ? LIMIT 1;

-- name: ListTopics :many
SELECT * FROM topics
ORDER BY name ASC;

-- name: InsertTopic :one
INSERT INTO topics (
    id,
    name,
    max_retries,
    base_interval_secs,
    max_interval_secs
) VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteAllTopics :exec
DELETE FROM topics;