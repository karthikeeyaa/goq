# gosub — Technical Design Specification

> A lightweight, high-performance, push-based Pub/Sub message broker and webhook delivery engine written in Go.
>
> *Made by Claude Sonnet 4.6 Medium — Version 1.0, June 2025*

---

## Table of Contents

1. [Overview](#1-overview)
2. [High-Level Architecture](#2-high-level-architecture)
3. [Fixture File (Deployment Contract)](#3-fixture-file-deployment-contract)
4. [Data Models & Database Schema](#4-data-models--database-schema)
5. [REST API Specification](#5-rest-api-specification)
6. [Delivery Guarantee & Reliability](#6-delivery-guarantee--reliability)
7. [Security & Signed Webhooks](#7-security--signed-webhooks)
8. [Technology Stack](#8-technology-stack)
9. [Example Application: DevStream](#9-example-application-devstream)
10. [Low-Level Implementation Notes](#10-low-level-implementation-notes)
11. [Future Considerations](#11-future-considerations)

---

## 1. Overview

`gosub` is a domain-agnostic, push-based Pub/Sub message broker and HTTP webhook delivery engine. It acts as a reliable intermediary between event producers and consumers, guaranteeing durable event delivery over HTTP with built-in retry logic, dead-letter queuing, and signed payload delivery.

`gosub` is a plug-and-play infrastructure primitive — it has no knowledge of business domains. The teams deploying gosub declare what topics and operations exist via a fixture file at deployment time. Everything else — who subscribes, where payloads are sent — is managed at runtime via the API.

### 1.1 Design Philosophy

- **Domain-agnostic**: gosub knows nothing about payments, invoices, or users. It moves any JSON payload reliably from A to B.
- **Fixture-driven topology**: Topics, operations, and their schemas are declared in a fixture file at deploy time — not hardcoded in source.
- **API-driven subscriptions**: Subscribers register at runtime via REST API. No restart required.
- **Fan-out by default**: Every operation supports multiple independent subscribers. Each gets its own delivery queue, retry state, and DLQ entry.
- **Guaranteed delivery**: Every message is persisted before acknowledgement. No in-memory-only queuing.

### 1.2 Key Terminology

| Term | Definition |
|---|---|
| **Topic** | A logical grouping of related operations (e.g. `payments`, `invoices`). Carries its own retry configuration. |
| **Operation** | A named event type within a topic (e.g. `payment.success`). Has a JSON schema that all published payloads must conform to. |
| **Subscription** | A registered consumer endpoint (URL) listening to a specific operation. One operation can have many subscriptions. |
| **Message** | A persisted payload published to a specific operation. Tracked through its full lifecycle. |
| **Delivery Attempt** | A single HTTP POST attempt to a subscriber URL. Tracked with status, response code, and retry schedule. |
| **DLQ** | Dead Letter Queue. Where messages land when all delivery attempts for a subscription are exhausted. |
| **Fixture** | A JSON file loaded at startup that declares all topics and operations gosub supports in this deployment. |

---

## 2. High-Level Architecture

gosub is composed of four primary runtime components:

| Component | Responsibility |
|---|---|
| **Ingestion Server** | HTTP server that accepts publish requests from producers, validates payloads against the operation schema, persists messages, and returns `202 Accepted`. |
| **Management Server** | HTTP server exposing the subscription CRUD API, DLQ inspection endpoints, and health checks. |
| **Dispatcher Worker Pool** | A pool of goroutines that polls the database for pending messages and dispatches them to registered subscriber URLs over HTTP. |
| **Persistent Store** | SQLite (default) or PostgreSQL. The source of truth for all messages, subscriptions, delivery attempts, and DLQ state. |

### 2.1 Message Flow

```
Producer
  │
  │  POST /api/v1/publish
  ▼
Ingestion Server
  │  Validate payload schema
  │  Persist message (status=pending)
  │  Return 202 Accepted
  ▼
Database (messages table)
  │
  │  Worker polls for pending messages
  ▼
Dispatcher Worker Pool
  │
  ├──▶ Subscription A  ──▶  HTTP POST  ──▶  Consumer A  (2xx → succeeded)
  │                                                      (non-2xx → retry → DLQ)
  │
  ├──▶ Subscription B  ──▶  HTTP POST  ──▶  Consumer B  (2xx → succeeded)
  │
  └──▶ Subscription C  ──▶  HTTP POST  ──▶  Consumer C  (non-2xx → retry → DLQ)
```

The complete lifecycle of a message through gosub:

1. Producer sends `POST /api/v1/publish` with an operation name and JSON payload.
2. Ingestion Server validates the payload against the JSON schema registered for that operation in the fixture.
3. On validation success, the message is persisted to the `messages` table with `status = pending`. A `message_id` is returned to the producer with HTTP `202 Accepted`.
4. The Dispatcher Worker Pool polls the database for messages with `status = pending`.
5. For each pending message, the worker looks up all active subscriptions for that operation. One delivery attempt record is created per subscription.
6. The worker signs the payload using HMAC-SHA256 with the subscription's secret key, then sends an HTTP POST to the consumer URL.
7. On `2xx` response: delivery attempt is marked `succeeded`. If all subscriptions for a message succeed, the message status becomes `succeeded`.
8. On non-`2xx` or timeout: delivery attempt is scheduled for retry using exponential backoff. Message status remains `processing`.
9. On max retries exceeded: delivery attempt is marked `dead`. An entry is written to the DLQ. Administrators can inspect and replay DLQ messages via API.

### 2.2 Fan-Out Model

When a message is published to an operation that has three active subscriptions, gosub creates three independent delivery attempt chains. Each subscription:

- Receives its own copy of the payload.
- Has its own retry counter, starting from 0.
- Can independently succeed, fail, or reach the DLQ.
- Uses the retry configuration of its parent topic.

> **Important:** A failure in delivering to Subscriber B does not affect delivery to Subscribers A or C. Each delivery chain is fully isolated.

---

## 3. Fixture File (Deployment Contract)

The fixture file is a JSON document loaded by gosub at startup. It declares the complete set of topics and operations this instance of gosub will accept. Think of it as the contract between gosub and the teams deploying it.

The fixture is not a runtime configuration file — it does not change without a redeployment. This is intentional: it creates a stable, auditable contract for what events gosub processes.

### 3.1 What the Fixture Declares

- All topics gosub supports, along with their retry configuration.
- All operations within each topic, along with their JSON schema for payload validation.

The fixture does **not** declare subscribers. Subscriptions are managed at runtime via the API. This separation means:

- Adding a new subscriber to an existing operation requires **no redeployment**.
- Adding a new operation or topic requires a fixture update and redeployment.

### 3.2 Fixture Schema

```json
{
  "topics": [
    {
      "name": "string",
      "retry_config": {
        "max_retries": "integer",
        "base_interval_seconds": "integer",
        "max_interval_seconds": "integer"
      },
      "operations": [
        {
          "name": "string",
          "schema": { }
        }
      ]
    }
  ]
}
```

### 3.3 Fixture Example

```json
{
  "topics": [
    {
      "name": "payments",
      "retry_config": {
        "max_retries": 5,
        "base_interval_seconds": 10,
        "max_interval_seconds": 3600
      },
      "operations": [
        {
          "name": "payment.success",
          "schema": {
            "type": "object",
            "required": ["id", "customer_id", "amount", "currency", "timestamp"],
            "properties": {
              "id":          { "type": "string" },
              "customer_id": { "type": "string" },
              "amount":      { "type": "integer" },
              "currency":    { "type": "string" },
              "timestamp":   { "type": "integer" }
            }
          }
        },
        {
          "name": "payment.failed",
          "schema": {
            "type": "object",
            "required": ["id", "customer_id", "amount", "currency", "failure_reason", "timestamp"],
            "properties": {
              "id":             { "type": "string" },
              "customer_id":    { "type": "string" },
              "amount":         { "type": "integer" },
              "currency":       { "type": "string" },
              "failure_reason": { "type": "string" },
              "timestamp":      { "type": "integer" }
            }
          }
        }
      ]
    }
  ]
}
```

### 3.4 Startup Behaviour

When gosub starts, it reads the fixture file and performs the following:

1. Parses and validates the fixture structure.
2. Upserts all declared topics into the `topics` table (insert or update on conflict).
3. Upserts all declared operations into the `operations` table.
4. Logs a warning for any topics/operations in the database that are no longer in the fixture (orphaned records are not deleted automatically — manual cleanup required).
5. Begins accepting traffic.

---

## 4. Data Models & Database Schema

gosub uses SQLite by default (zero-dependency embedded deployment) with the schema designed to be portable to PostgreSQL.

### 4.1 Schema Overview

| Table | Purpose |
|---|---|
| `topics` | One row per topic declared in the fixture. Holds retry configuration. |
| `operations` | One row per operation, linked to its parent topic. |
| `subscriptions` | Runtime-registered consumer endpoints. One subscription = one operation + one consumer URL. |
| `messages` | Every published payload, with its full lifecycle status. |
| `delivery_attempts` | Each HTTP dispatch attempt per subscription per message, with retry schedule. |
| `dlq` | Terminal resting place for messages that exhausted all retries for a given subscription. |

### 4.2 SQL Definitions

#### Topics

```sql
CREATE TABLE topics (
    name                 TEXT    PRIMARY KEY,
    max_retries          INTEGER NOT NULL DEFAULT 5,
    base_interval_secs   INTEGER NOT NULL DEFAULT 10,
    max_interval_secs    INTEGER NOT NULL DEFAULT 3600,
    created_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### Operations

```sql
CREATE TABLE operations (
    name        TEXT PRIMARY KEY,
    topic_name  TEXT NOT NULL REFERENCES topics(name) ON DELETE CASCADE,
    schema_json TEXT NOT NULL,   -- JSON Schema string for payload validation
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

#### Subscriptions

```sql
CREATE TABLE subscriptions (
    id              TEXT    PRIMARY KEY,  -- UUIDv4
    operation_name  TEXT    NOT NULL REFERENCES operations(name) ON DELETE CASCADE,
    consumer_url    TEXT    NOT NULL,
    secret_key      TEXT    NOT NULL,     -- HMAC signing key for this subscription
    is_active       INTEGER NOT NULL DEFAULT 1,  -- 0 = paused, 1 = active
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX idx_subscriptions_unique
    ON subscriptions(operation_name, consumer_url);
```

#### Messages

```sql
CREATE TABLE messages (
    id              TEXT PRIMARY KEY,  -- UUIDv4
    topic_name      TEXT NOT NULL REFERENCES topics(name) ON DELETE CASCADE,
    operation_name  TEXT NOT NULL REFERENCES operations(name) ON DELETE CASCADE,
    payload         TEXT NOT NULL,     -- Raw JSON string
    status          TEXT NOT NULL DEFAULT 'pending',
                    -- pending | processing | succeeded | partially_failed | failed
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_polling
    ON messages(topic_name, status, created_at);
```

#### Delivery Attempts

```sql
CREATE TABLE delivery_attempts (
    id               TEXT PRIMARY KEY,  -- UUIDv4
    message_id       TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    subscription_id  TEXT NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
    attempt_number   INTEGER NOT NULL DEFAULT 1,
    status_code      INTEGER,           -- HTTP response code received
    error_message    TEXT,              -- Network error or timeout description
    next_retry_at    DATETIME,          -- NULL if terminal
    status           TEXT NOT NULL,
                     -- pending | in_flight | succeeded | failed | dead
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_attempts_polling
    ON delivery_attempts(status, next_retry_at);
```

#### Dead Letter Queue

```sql
CREATE TABLE dlq (
    id               TEXT PRIMARY KEY,  -- UUIDv4
    message_id       TEXT NOT NULL REFERENCES messages(id),
    subscription_id  TEXT NOT NULL REFERENCES subscriptions(id),
    last_attempt_id  TEXT NOT NULL REFERENCES delivery_attempts(id),
    reason           TEXT NOT NULL,     -- Human-readable failure reason
    replayed_at      DATETIME,          -- Set when admin triggers a replay
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### 4.3 Message Status Lifecycle

| Status | Meaning |
|---|---|
| `pending` | Newly persisted. Not yet picked up by any worker. |
| `processing` | At least one delivery attempt is in flight or scheduled for retry. |
| `succeeded` | All subscriptions for this message have been delivered successfully. |
| `partially_failed` | Some subscriptions succeeded; at least one is in DLQ. |
| `failed` | All subscriptions exhausted retries and are in DLQ. |

---

## 5. REST API Specification

gosub exposes two groups of HTTP endpoints. All requests and responses use JSON. All endpoints are versioned under `/api/v1`.

### 5.1 Publish API

#### `POST /api/v1/publish`

Accepts an event from a producer. Validates the payload and persists the message.

| Field | Value |
|---|---|
| Method | `POST` |
| Path | `/api/v1/publish` |
| Content-Type | `application/json` |
| Auth | Optional — API key via `X-Gosub-Api-Key` header |

**Request Body**

```json
{
  "operation": "payment.success",
  "payload": {
    "id": "pay_abc123",
    "customer_id": "cust_xyz",
    "amount": 4999,
    "currency": "USD",
    "timestamp": 1718000000
  }
}
```

**Responses**

| Status | Condition | Response Body |
|---|---|---|
| `202 Accepted` | Message persisted and queued. | `{ "message_id": "uuid", "status": "queued" }` |
| `400 Bad Request` | Payload fails JSON schema validation. | `{ "error": "validation_failed", "details": [...] }` |
| `404 Not Found` | Operation name not found in fixture. | `{ "error": "operation_not_found" }` |
| `422 Unprocessable` | No active subscriptions for this operation. | `{ "error": "no_active_subscriptions" }` |

---

### 5.2 Subscription Management API

#### `POST /api/v1/subscriptions`

Register a new subscriber for an operation.

```json
{
  "operation_name": "payment.success",
  "consumer_url": "https://internal-service/webhooks/payments",
  "secret_key": "your-hmac-signing-secret"
}
```

| Status | Condition |
|---|---|
| `201 Created` | Subscription registered. Body contains `subscription_id`. |
| `409 Conflict` | A subscription for this `operation + consumer_url` already exists. |
| `404 Not Found` | Operation does not exist in fixture. |

---

#### `GET /api/v1/subscriptions`

List all subscriptions, optionally filtered.

| Query Param | Type | Description |
|---|---|---|
| `operation_name` | string (optional) | Filter subscriptions by operation. |
| `is_active` | boolean (optional) | Filter by active/paused state. |

---

#### `PATCH /api/v1/subscriptions/:id`

Update a subscription. Supports pausing (`is_active: false`) or updating the consumer URL.

---

#### `DELETE /api/v1/subscriptions/:id`

Remove a subscription permanently. In-flight delivery attempts for this subscription are allowed to complete.

---

### 5.3 Dead Letter Queue API

#### `GET /api/v1/dlq`

List entries in the DLQ. Supports filtering by `subscription_id` or `message_id`.

#### `POST /api/v1/dlq/:id/replay`

Triggers a re-delivery for a specific DLQ entry. Creates a fresh delivery attempt record and re-enqueues the original payload. The replay counter starts fresh — `max_retries` applies again.

---

### 5.4 Health & Observability

| Endpoint | Description |
|---|---|
| `GET /health` | Liveness check. Returns `200 OK` if the server is running. |
| `GET /ready` | Readiness check. Returns `200 OK` only if the database connection is alive. |
| `GET /api/v1/metrics` | Prometheus-compatible metrics: messages queued, delivered, failed, DLQ count, worker pool utilisation. |

---

## 6. Delivery Guarantee & Reliability

### 6.1 Exponential Backoff

When a delivery attempt fails (non-2xx response or network timeout), gosub schedules a retry using exponential backoff with jitter:

```
delay = min(max_interval, base_interval × 2^(attempt_number - 1)) + jitter

Where jitter is a random value in [0, base_interval] to prevent thundering herd.
```

Example retry schedule with defaults (`base=10s`, `max=3600s`, `max_retries=5`):

| Attempt | Base Delay | With Jitter (approx) |
|---|---|---|
| 1 | 10s | 10–20s |
| 2 | 20s | 20–30s |
| 3 | 40s | 40–50s |
| 4 | 80s | 80–90s |
| 5 (final) | 160s | 160–170s |

### 6.2 Dead Letter Queue (DLQ)

When the final retry attempt fails, gosub:

1. Sets `delivery_attempts.status` to `dead`.
2. Creates a record in the `dlq` table linking the message and subscription.
3. Updates `messages.status` to `failed` (or `partially_failed` if other subscriptions succeeded).
4. Emits a `dlq.entry_created` internal metric/event (can be wired to alerting).

DLQ entries are permanent until an administrator either replays or manually deletes them. Replay creates a fresh delivery chain with a full retry budget.

### 6.3 At-Least-Once Delivery

gosub guarantees **at-least-once** delivery, not exactly-once. This means:

- Every message that is persisted will be delivered at least once to every active subscriber.
- In rare cases (worker crash after HTTP success but before DB update), a message may be delivered more than once.
- **Consumers must be idempotent** — they should handle duplicate deliveries gracefully using the `X-Gosub-Message-Id` header included in every HTTP dispatch.

### 6.4 Worker Concurrency

The dispatcher uses a configurable pool of goroutines. Each worker independently polls for pending delivery attempts. A semaphore pattern limits concurrent outgoing HTTP connections per subscription, preventing gosub from overwhelming slow consumers.

```go
// Pseudocode
type Dispatcher struct {
    db          *sql.DB
    httpClient  *http.Client
    workerCount int                              // Default: 10
    semaphores  map[string]*semaphore.Weighted   // per subscription_id
}
```

---

## 7. Security & Signed Webhooks

### 7.1 Outgoing Request Headers

Every HTTP POST dispatched by gosub includes the following headers:

| Header | Value | Purpose |
|---|---|---|
| `X-Gosub-Message-Id` | UUIDv4 of the message | Consumer idempotency key. |
| `X-Gosub-Operation` | e.g. `payment.success` | Which operation this event belongs to. |
| `X-Gosub-Timestamp` | Unix timestamp (seconds) | Time of dispatch. Used to prevent replay attacks. |
| `X-Gosub-Signature` | HMAC-SHA256 hex string | Payload integrity signature. |
| `X-Gosub-Attempt` | Integer | Which attempt number this is (1 = first try). |

### 7.2 Signature Algorithm

The signature is computed over a deterministic string combining the dispatch timestamp and the raw payload body:

```go
package signature

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
)

func Generate(payload []byte, timestamp string, secretKey string) string {
    mac := hmac.New(sha256.New, []byte(secretKey))
    data := fmt.Sprintf("%s.%s", timestamp, string(payload))
    mac.Write([]byte(data))
    return hex.EncodeToString(mac.Sum(nil))
}
```

### 7.3 Consumer Validation Flow

1. Read `X-Gosub-Timestamp`. If it is more than 5 minutes old, reject the request (replay attack prevention).
2. Read `X-Gosub-Signature` from the header.
3. Re-compute the signature locally using your secret key, the raw request body, and the timestamp.
4. Compare using constant-time comparison (`crypto/subtle.ConstantTimeCompare`) to prevent timing attacks.
5. If signatures match, the payload is authentic.

---

## 8. Technology Stack

| Concern | Choice | Rationale |
|---|---|---|
| Language | Go 1.22+ | Excellent concurrency primitives (goroutines, channels). Low memory footprint. Ideal for a high-throughput worker pool. |
| HTTP Router | `github.com/go-chi/chi/v5` | Lightweight, stdlib-compliant. No magic. Easy middleware composition. |
| Database (default) | SQLite via `github.com/mattn/go-sqlite3` | Zero-dependency embedded deployment. Single binary. Sufficient for moderate throughput. |
| Database (scale) | PostgreSQL via `lib/pq` | For high-throughput or multi-instance deployments. Schema is compatible. |
| JSON Validation | `github.com/xeipuuv/gojsonschema` | Full JSON Schema Draft-04 support. Used to validate payloads against fixture-defined schemas. |
| Concurrency | `golang.org/x/sync/semaphore` | Per-subscription connection cap to prevent overwhelming slow consumers. |
| Config | Environment variables + fixture JSON | Twelve-factor compliant. No vendor-specific config system. |

### 8.1 Project Structure

```
gosub/
├── cmd/
│   └── gosub/          # main.go — entry point
├── internal/
│   ├── fixture/        # Fixture file parsing and DB upsert
│   ├── api/            # HTTP handlers (publish, subscriptions, DLQ)
│   ├── dispatcher/     # Worker pool and delivery logic
│   ├── db/             # SQL queries and migrations
│   ├── signature/      # HMAC generation and verification
│   └── schema/         # JSON Schema validation wrapper
├── migrations/         # SQL migration files
├── fixture.json        # Default fixture (mounted at deploy time)
└── config.env.example  # Environment variable reference
```

### 8.2 Configuration (Environment Variables)

| Variable | Default | Description |
|---|---|---|
| `GOSUB_DB_DRIVER` | `sqlite3` | Database driver: `sqlite3` or `postgres`. |
| `GOSUB_DB_DSN` | `./gosub.db` | Database connection string or file path. |
| `GOSUB_FIXTURE_PATH` | `./fixture.json` | Path to the fixture file. |
| `GOSUB_WORKER_COUNT` | `10` | Number of dispatcher goroutines. |
| `GOSUB_POLL_INTERVAL_MS` | `500` | How often workers poll the DB for pending messages (ms). |
| `GOSUB_HTTP_TIMEOUT_SECONDS` | `30` | Timeout for outgoing HTTP dispatch requests. |
| `GOSUB_PORT` | `8080` | Port for the HTTP server. |
| `GOSUB_API_KEY` | *(empty)* | If set, all API requests must include `X-Gosub-Api-Key`. |

---

## 9. Example Application: DevStream

To demonstrate gosub in an internal platform context, **DevStream** is a thin internal developer platform event bus used by an engineering team's own infrastructure services.

Whenever a meaningful engineering event happens — a deployment, a build, a feature flag toggle — DevStream publishes it to gosub. Multiple internal tools subscribe to receive those events and act on them independently.

This is a pure internal use case. No external clients, no payment webhooks, no SaaS integrations. Everything stays within the company's infrastructure.

### 9.1 Topics & Operations

| Topic | Operations | Schema Fields |
|---|---|---|
| `deployments` | `deployment.started` | `service_name`, `version`, `environment`, `deployed_by`, `timestamp` |
| `deployments` | `deployment.succeeded` | `service_name`, `version`, `environment`, `duration_seconds`, `timestamp` |
| `deployments` | `deployment.failed` | `service_name`, `version`, `environment`, `error_message`, `timestamp` |
| `builds` | `build.triggered` | `repo`, `branch`, `commit_sha`, `triggered_by`, `timestamp` |
| `builds` | `build.completed` | `repo`, `branch`, `commit_sha`, `status`, `duration_seconds`, `timestamp` |
| `feature_flags` | `flag.toggled` | `flag_key`, `previous_value`, `new_value`, `changed_by`, `timestamp` |
| `incidents` | `incident.opened` | `id`, `title`, `severity` (P1–P4), `service_name`, `timestamp` |
| `incidents` | `incident.resolved` | `id`, `resolution_summary`, `duration_minutes`, `timestamp` |

### 9.2 Subscribers

| Subscriber Service | Listens To | What It Does |
|---|---|---|
| Slack Notifier | `deployment.*`, `build.completed`, `incident.*` | Posts formatted messages to team Slack channels. |
| Metrics Aggregator | `deployment.succeeded`, `build.completed` | Records deploy frequency and build duration for DORA metrics dashboard. |
| On-call Pager | `incident.opened` (severity P1, P2) | Triggers PagerDuty escalation for high-severity incidents. |
| Audit Logger | All operations | Writes every event to an immutable append-only audit store. |
| Cache Invalidator | `flag.toggled` | Flushes feature flag caches across all running service instances. |
| Rollback Guard | `deployment.failed` | Automatically triggers a rollback pipeline if the failed deployment was to production. |

### 9.3 Why This Demonstrates gosub Well

- **Fan-out in practice**: a single `deployment.failed` event triggers three independent subscribers (Slack Notifier, Audit Logger, Rollback Guard), each with their own delivery and retry state.
- **Fixture as contract**: the DevStream team declares all topics and operations in `fixture.json` before deploying. gosub enforces schema compliance on every publish call.
- **No domain coupling**: gosub has no idea what a "deployment" is. It just validates JSON and delivers reliably.
- **DLQ value**: if the Rollback Guard is temporarily down during a failed deployment, gosub holds the event in DLQ. Once Rollback Guard recovers, the admin replays the entry and the rollback still happens.
- **Internal-only**: no external clients, no public-facing webhooks, no compliance overhead from external data sharing.

---

## 10. Low-Level Implementation Notes

### 10.1 Dispatcher Poll Loop

The dispatcher runs a tight poll loop per worker goroutine. Each iteration:

1. Queries `delivery_attempts WHERE status = 'pending' AND next_retry_at <= NOW() ORDER BY next_retry_at ASC LIMIT 1`.
2. Atomically updates the row to `status = 'in_flight'` (prevents double-dispatch by concurrent workers).
3. Fetches the linked message and subscription.
4. Executes the HTTP dispatch.
5. On success: updates attempt to `succeeded`. Checks if all attempts for the message are done.
6. On failure: computes `next_retry_at`, increments `attempt_number`. If at `max_retries`, moves to DLQ.

> **Concurrency Safety:** The atomic status transition from `pending` to `in_flight` must be done in a single `UPDATE ... WHERE status = 'pending'` statement. This prevents two workers from picking up the same delivery attempt.

### 10.2 Schema Validation

On every publish request, gosub runs the incoming payload against the JSON Schema stored in `operations.schema_json` using `gojsonschema`. Validation errors are returned as structured `400` responses including the specific field and constraint that failed.

### 10.3 Database Migrations

gosub uses sequential SQL migration files in the `/migrations` directory. On startup, gosub checks which migrations have been applied and runs any pending ones in order. This keeps the schema in sync without requiring an external migration tool.

### 10.4 HTTP Client Configuration

The shared HTTP client used for dispatch should be configured with:

- **Timeout**: `GOSUB_HTTP_TIMEOUT_SECONDS` (default 30s).
- **Keep-alive enabled** — reuse connections for the same consumer host.
- **No automatic redirect following** — a `301/302` from a consumer counts as a failed delivery, not a redirect.

### 10.5 Graceful Shutdown

On `SIGTERM`, gosub:

1. Stops accepting new publish requests (returns `503`).
2. Signals workers to finish their current in-flight dispatch and not pick up new work.
3. Waits up to 30 seconds for all in-flight dispatches to complete.
4. Closes the database connection and exits.

Any `delivery_attempts` that were marked `in_flight` but not completed will be reset to `pending` on next startup (handled by a startup cleanup query).

---

## 11. Future Considerations

The following are out of scope for v1 but worth designing for later:

| Feature | Description |
|---|---|
| **Subscription filters** | Allow subscribers to declare a JSONPath filter — e.g. only receive `payment.success` events where `payload.amount > 10000`. |
| **Multi-instance deployment** | Distributed worker coordination using PostgreSQL advisory locks or a Redis-based claim mechanism, enabling horizontal scaling. |
| **Event ordering guarantees** | Per-topic FIFO ordering for use cases where event sequence matters (e.g. state machine transitions). |
| **Admin UI** | A simple web dashboard for browsing messages, subscriptions, and DLQ entries — with replay buttons. |
| **Webhook rate limiting** | Per-subscription rate cap (e.g. max 100 req/min to a single consumer URL) to protect slow consumers. |
| **Mutual TLS** | mTLS support for dispatch requests to consumers that require client certificate authentication. |

---

*Made by Antigravity*