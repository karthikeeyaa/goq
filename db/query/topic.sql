-- name: GetTopic :one
SELECT *
FROM topics
WHERE name = ?
LIMIT 1;

-- name: UpsertTopic :exec
INSERT INTO topics (name, mode, retention_seconds, schema_validation, schema_json)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
    mode = excluded.mode,
    retention_seconds = excluded.retention_seconds,
    schema_validation = excluded.schema_validation,
    schema_json = excluded.schema_json,
    updated_at = CURRENT_TIMESTAMP;