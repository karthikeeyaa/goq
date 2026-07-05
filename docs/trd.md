*Written by an `earthling` basis of `technical_spec.md`*

# Technical Specification: gosub

## Requirements

* A lightweight, log-based message broker for internal microservice integration — domain-agnostic, plug-and-play.
* Producers publish messages to a topic.
* Consumers pull messages from a topic using offsets, similar to Kafka's fetch model.
* Every topic represents an independent, append-only message log.
* Each topic has a unique identifier used by producers and consumers.
* Payloads are stored as binary (BLOB). Schema validation is optional per topic and configured in the fixture.
* Messages are retained for a configurable duration before being archived and removed from the active SQLite store.
* Consumers track their own offset client-side and pass it on each pull request. gosub does not verify or persist consumer identity in Phase 1.
* If no offset is provided while pulling, gosub returns the current head offset with zero messages, allowing the consumer to start watching from "now."
* Topics are declared only via the fixture file at startup. No dynamic topic creation via API.

### Flow Breakdown

1. **Publish Event**: The producer sends a JSON payload to `gosub` targeting a specific topic.
2. **Validation (optional)**: If `schema_validation` is enabled for the topic, the payload is validated against the topic's `schema_json`. If disabled, the payload is stored as-is.
3. **Persistence**: `gosub` appends the message to the SQLite message log with an auto-incrementing `id`.
4. **Producer Acknowledgment**: The system responds with HTTP `202 Accepted`, including the assigned `id`.
5. **Pull Request**: A consumer requests messages using a topic name, last known offset (`id`), and batch limit.
6. **Message Retrieval**: `gosub` returns all messages in that topic with `id` greater than the supplied offset, ordered ascending, up to the limit.
7. **Retention**: A background retention worker periodically archives expired messages to disk and removes them from the active SQLite store.

---

## Low Level

1. Golang
2. SQLite (sqlc) — `modernc.org/sqlite`
3. Chi router — `go.chi.dev/chi/v5`
4. HTTP API Server implementing publish and pull endpoints with standard Chi middleware.
5. Database schema and migrations using SQLite with automatic startup migrations.
6. sqlc for type-safe database access — no ORM.
7. Offset-based message retrieval using indexed sequential queries on `(topic_name, id)`.
8. Binary payload storage using SQLite BLOB columns.
9. Configurable retention per topic, declared in the fixture.
10. Background Retention Worker responsible for:
    * archiving expired messages to append-only files on disk
    * deleting archived messages from the active SQLite store
11. Archive files are generated dynamically per topic at archival time (not pre-declared in the fixture). Format and chunking strategy TBD at implementation time — recommend newline-delimited binary records, file-per-time-window, named by offset range for fast lookup during replay.

### Message ID / Offset Design

`id` and `topic_name` together form the logical identity of a message (similar to Kafka's `offset` + `partition`). Two implementation paths exist:

* **Path A (recommended) — global auto-increment.** `id` is a single-column `INTEGER PRIMARY KEY AUTOINCREMENT` on the `messages` table, unique across all topics. A `payments` message and an `invoices` message will never share the same `id` by coincidence, but `id` values are not guaranteed to start at 1 per topic — they're simply unique and increasing overall. No extra `SELECT` needed before insert; SQLite handles this internally with no race condition risk, since the driver is configured with a single writer connection (`SetMaxOpenConns(1)`).
* **Path B — per-topic counter.** `id` restarts at 1 for each topic, matching Kafka's per-partition offset semantics exactly. Requires either a `SELECT MAX(id)+1 WHERE topic_name = ?` before each insert, or a separate counter row maintained per topic. Slightly more SQL overhead per write, but offsets are more intuitive when inspected per-topic (e.g. "invoices is at offset 42" vs "invoices is at offset 91841203").

Given the single-writer-connection setup already in place, Path A carries negligible practical downside and removes a moving part. Default to **Path A** unless a concrete reason for per-topic-zero-based offsets emerges during implementation.

```sql
CREATE TABLE messages (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    topic_name   TEXT NOT NULL,
    payload      BLOB NOT NULL,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_topic_id ON messages(topic_name, id);
```

### Pull Semantics

```
GET /api/v1/pull/{topic}?from={id}&limit={n}
```

* `from` omitted → gosub returns `{ "messages": [], "head_offset": <current max id for topic> }`. Consumer stores `head_offset` and uses it as `from` on the next call to receive only new messages going forward.
* `from=earliest` → gosub resolves to the lowest `id` still available for that topic (in SQLite, or via archive lookup if pruned).
* `from={id}` → gosub returns messages with `id > {id}`, ascending, up to `limit`.

---

## Phase 2 (Reliability Layer)

A separate worker service can be enabled to add push-based delivery capabilities on top of the same message log.

### 1. Push Delivery
* HTTP webhook subscriptions, configured per topic in the fixture.
* Background dispatcher worker pool.
* HMAC request signing per subscription.
* Retry with exponential backoff, configurable per topic.
* Dead Letter Queue (DLQ) for exhausted retries, with manual replay support.
* Fan-out by default: every active subscription on a topic receives every message, with fully independent delivery and retry state per subscription.

### 2. Consumer Groups & Offset Tracking
Adds Kafka-style server-side offset persistence so pull consumers can resume after a restart without remembering their own position, and so multiple instances of the same logical consumer can coordinate.

Proposed single-table design:

```sql
CREATE TABLE consumer_groups (
    group_id      TEXT NOT NULL,
    topic_name    TEXT NOT NULL,
    api_key       TEXT NOT NULL,       -- identifies and authenticates the group
    last_offset   INTEGER NOT NULL DEFAULT 0,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, topic_name)
);
```

* `group_id` — a consumer-chosen identifier (e.g. `"billing-service"`). Multiple processes can share the same `group_id` to behave as one logical consumer (their offset advances together, enabling competing-consumer semantics later if needed).
* `api_key` — lightweight auth so one group can't read or advance another group's offset by guessing its `group_id`.
* `last_offset` — updated only when the consumer explicitly commits (e.g. `POST /pull/{topic}/commit`), not on every fetch. This mirrors Kafka's separation of "fetch" from "commit" — a consumer can read messages speculatively without advancing its committed position until it's confident they were processed.

This table is intentionally deferred to Phase 2 — Phase 1 pull consumers remain fully stateless from gosub's perspective, tracking their own offset client-side with zero server-side bookkeeping or lookup cost.

### 3. Replay from Archive
* Pull requests with `from` values below the SQLite retention window transparently fall through to archive files.
* Enables full historical replay within whatever archive retention policy is configured, independent of the hot-path retention window.

---