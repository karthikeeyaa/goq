-- name: GetOperation :one
SELECT * FROM operations
WHERE id = ? LIMIT 1;

-- name: GetOperationByName :one
SELECT * FROM operations
WHERE name = ? LIMIT 1;

-- name: ListOperations :many
SELECT * FROM operations
ORDER BY name ASC;

-- name: ListOperationsByTopic :many
SELECT * FROM operations
WHERE topic_id = ?
ORDER BY name ASC;

-- name: InsertOperation :one
INSERT INTO operations (
    id,
    topic_id,
    name,
    schema_json
) VALUES (?, ?, ?, ?)
RETURNING *;

-- name: DeleteAllOperations :exec
DELETE FROM operations;