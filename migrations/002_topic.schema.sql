-- 002_topic.schema.sql

CREATE TABLE topic (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    retention_ms INTEGER DEFAULT 86400000, -- 
    cleanup_policy TEXT DEFAULT 'delete' CHECK (cleanup_policy IN ('delete', 'compact')),
    max_message_bytes INTEGER DEFAULT 1048576, -- 1MB
    log_index_interval_bytes INTEGER DEFAULT 4096,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);