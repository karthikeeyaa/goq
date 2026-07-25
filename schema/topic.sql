-- schema.sql
-- Database schema for gosub

CREATE TABLE topic (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    retention_ms INTEGER DEFAULT 86400000,
    cleanup_policy TEXT DEFAULT 'delete' CHECK (cleanup_policy IN ('delete', 'compact')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);