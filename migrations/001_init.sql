-- 001_init.sql
-- Database schema for gosub

CREATE TABLE topics (
    name TEXT PRIMARY KEY,
    mode TEXT NOT NULL DEFAULT 'pull'
        CHECK (mode IN ('pull', 'push')),
    retention_seconds INTEGER NOT NULL CHECK (retention_seconds > 0),
    schema_validation INTEGER NOT NULL DEFAULT 0
        CHECK (schema_validation IN (0, 1)),
    schema_json TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE messages (
    offset       INTEGER PRIMARY KEY AUTOINCREMENT,
    topic_name   TEXT NOT NULL,
    payload      BLOB NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (topic_name) REFERENCES topics(name) ON DELETE CASCADE
);

CREATE INDEX idx_messages_topic_offset
ON messages(topic_name, offset);

CREATE INDEX idx_messages_created_at
ON messages(created_at);