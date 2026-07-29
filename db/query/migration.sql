-- name: CreateGoqMigrationTable :exec
CREATE TABLE IF NOT EXISTS goq_migrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    hash TEXT NOT NULL,
    applied DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- name: GetGoqMigration :one
SELECT *
FROM goq_migrations
WHERE name = ?
LIMIT 1;

-- name: UpsertGoqMigration :exec
INSERT INTO goq_migrations (name, hash)
VALUES (?, ?)
ON CONFLICT(name) DO UPDATE SET
    hash = excluded.hash, 
    applied = CURRENT_TIMESTAMP;