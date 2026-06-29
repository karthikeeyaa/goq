*Written by an `earthling` basis of `technical_spec.md`*


# Technical Specification: gosub

## Requirements

* A lightweight subscription-based message broker inspired by log-based messaging systems.
* Producers publish messages to a topic.
* Consumers pull messages from a topic using offsets.
* Every topic represents an independent append-only message log.
* Each topic has a unique identifier used by producers and consumers.
* Payloads are type-safe. Invalid payloads are rejected during ingestion.
* Messages are retained for a configurable duration before being archived and removed from the active store.
* Consumers maintain their own offsets and can replay messages by requesting an earlier offset.
* If no offset is provided while pulling, the topic's configured latest offset is used.

### Flow Breakdown

1. **Publish Event**: The producer sends a type-safe JSON payload to `gosub` targeting a specific topic.
2. **Validation**: The payload is validated against the topic's configured JSON schema.
3. **Persistence**: `gosub` appends the message to the SQLite message log.
4. **Producer Acknowledgment**: The system responds with HTTP `202 Accepted`.
5. **Pull Request**: A consumer requests messages using a topic name, last processed offset, and batch limit.
6. **Message Retrieval**: `gosub` returns all messages whose offsets are greater than the supplied offset.
7. **Retention**: A background retention worker periodically archives expired messages to disk and removes them from the active SQLite store.

---

## Low level

1. Golang
2. SQLite (sqlc) - `modernc.org/sqlite`
3. Chi router - `go.chi.dev/chi/v5`
5. HTTP API Server implementing publish, pull, and topic management endpoints with standard Chi middleware.
6. Database Schema & Migrations using SQLite with automatic startup migrations.
7. SQLC for type-safe database access.
8. Offset-based message retrieval using indexed sequential queries on `(topic_name, offset)`.
9. Background Retention Worker responsible for:

   * archiving expired messages to append-only files
   * deleting archived messages from SQLite
   * updating topic metadata where required
10. Binary payload storage using SQLite BLOB columns to avoid unnecessary serialization.
11. Configurable retention per topic.
12. Archive files stored per topic for long-term persistence and optional offline replay.

---

## Phase 2 (Reliability Layer)

A separate worker service can be enabled to add push-based delivery capabilities.

Features:

1. Push

   * HTTP webhook subscriptions
   * Background dispatcher
   * HMAC request signing
   * Retry with exponential backoff
   * Dead Letter Queue (DLQ)

2. Replay from archived messages



Post that kinda triggered me: https://x.com/ChShersh/status/2071531589972922686