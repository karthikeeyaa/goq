-- 001_init.sql
-- Database schema for gosub

CREATE TABLE topics (
    name                 TEXT    PRIMARY KEY,
    max_retries          INTEGER NOT NULL DEFAULT 5,
    base_interval_secs   INTEGER NOT NULL DEFAULT 10,
    max_interval_secs    INTEGER NOT NULL DEFAULT 3600,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE operations (
    name        TEXT PRIMARY KEY,
    topic_name  TEXT NOT NULL REFERENCES topics(name) ON DELETE CASCADE,
    schema_json TEXT NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE subscriptions (
    id              TEXT    PRIMARY KEY,
    operation_name  TEXT    NOT NULL REFERENCES operations(name) ON DELETE CASCADE,
    consumer_url    TEXT    NOT NULL,
    secret_key      TEXT    NOT NULL,
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_subscriptions_unique
    ON subscriptions(operation_name, consumer_url);

CREATE TABLE messages (
    id              TEXT PRIMARY KEY,
    topic_name      TEXT NOT NULL REFERENCES topics(name) ON DELETE CASCADE,
    operation_name  TEXT NOT NULL REFERENCES operations(name) ON DELETE CASCADE,
    payload         TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_polling
    ON messages(topic_name, status, created_at);

CREATE TABLE delivery_attempts (
    id               TEXT PRIMARY KEY,
    message_id       TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    subscription_id  TEXT NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    attempt_number   INTEGER NOT NULL DEFAULT 1,
    status_code      INTEGER,
    error_message    TEXT,
    next_retry_at    DATETIME,
    status           TEXT NOT NULL,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_attempts_polling
    ON delivery_attempts(status, next_retry_at);

CREATE TABLE dlq (
    id               TEXT PRIMARY KEY,
    message_id       TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    subscription_id  TEXT NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    last_attempt_id  TEXT NOT NULL REFERENCES delivery_attempts(id) ON DELETE CASCADE,
    reason           TEXT NOT NULL,
    replayed_at      DATETIME,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
