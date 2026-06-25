-- name: GetOperation :one
SELECT * FROM operations
WHERE name = ? LIMIT 1;

-- name: ListOperations :many
SELECT * FROM operations
ORDER BY name ASC;

-- name: ListOperationsByTopic :many
SELECT * FROM operations
WHERE topic_name = ?
ORDER BY name ASC;

-- name: UpsertOperation :one
INSERT INTO operations (
    name,
    topic_name,
    schema_json
) VALUES (?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
    topic_name = excluded.topic_name,
    schema_json = excluded.schema_json
RETURNING *;

-- name: DeleteOperation :exec
DELETE FROM operations
WHERE name = ?;
