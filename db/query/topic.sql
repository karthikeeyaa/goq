-- name: GetTopic :one
SELECT *
FROM topic
WHERE name = ?
LIMIT 1;

-- name: UpsertTopic :exec
INSERT INTO topic (name, retention_ms, cleanup_policy)
VALUES (?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
    retention_ms = excluded.retention_ms,
    cleanup_policy = excluded.cleanup_policy;