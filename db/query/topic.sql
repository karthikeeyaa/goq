-- name: GetTopic :one
SELECT * FROM topics
WHERE name = ? LIMIT 1;

-- name: ListTopics :many
SELECT * FROM topics
ORDER BY name ASC;

-- name: UpsertTopic :one
INSERT INTO topics (
    name,
    max_retries,
    base_interval_secs,
    max_interval_secs
) VALUES (?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
    max_retries = excluded.max_retries,
    base_interval_secs = excluded.base_interval_secs,
    max_interval_secs = excluded.max_interval_secs
RETURNING *;

-- name: DeleteTopic :exec
DELETE FROM topics
WHERE name = ?;
