-- name: GetTopic :one
SELECT *
FROM topic
WHERE name = ?
LIMIT 1;

-- name: ListTopics :many
SELECT *
FROM topic;

-- name: UpsertTopic :exec
INSERT INTO topic (
    name, 
    retention_ms, 
    cleanup_policy, 
    max_message_bytes, 
    log_index_interval_bytes
)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
    retention_ms = excluded.retention_ms,
    cleanup_policy = excluded.cleanup_policy,
    max_message_bytes = excluded.max_message_bytes,
    log_index_interval_bytes = excluded.log_index_interval_bytes;