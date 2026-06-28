-- 001_init.sql
-- Database schema for gosub

CREATE TABLE topics (
    id                   UUID    PRIMARY KEY,
    name                 TEXT    NOT NULL UNIQUE,
    max_retries          INTEGER NOT NULL DEFAULT 5,
    base_interval_secs   INTEGER NOT NULL DEFAULT 10,
    max_interval_secs    INTEGER NOT NULL DEFAULT 3600,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE operations (
    id          UUID PRIMARY KEY,
    topic_id    UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    name        TEXT NOT NULL UNIQUE,
    schema_json TEXT NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE subscriptions (
    id              UUID    PRIMARY KEY,
    topic_id        UUID    NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    consumer_url    TEXT    NOT NULL,
    secret_key      TEXT    NOT NULL,
    is_active       INTEGER NOT NULL DEFAULT 1,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_subscriptions_unique
    ON subscriptions(topic_id, consumer_url);

CREATE TABLE messages (
    id              UUID PRIMARY KEY,
    topic_id        UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    operation_id    UUID NOT NULL REFERENCES operations(id) ON DELETE CASCADE,
    payload         TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_polling
    ON messages(topic_id, status, created_at);