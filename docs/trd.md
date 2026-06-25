*Written by an `earthling` basis of `technical_spec.md`*


# Technical Specification: gosub

## Requirements

- A subscription based producer-consumer message queue service.
- Producer will publish the message to the queue, consumer will take the messages and send to subscribers webhook url
- Every topic will have a separate queue. There can be multiple topics
- Each topic have an identifier using which publisher will send messages
- Every topic have a subscriber
- Payload in each operation is type safe, mismatch in payload when posting will cause error.
- Messages sent to subscriber will have retry mechanism, exponential backoff. Crossing max retries will send the message to DLQ

### Flow Breakdown
1. **Publish Event**: The producer sends a type-safe JSON payload to `gosub` targeting a specific **Operation**.
2. **Persistence**: `gosub` persists the message to the database immediately to prevent loss.
3. **Producer Acknowledgment**: The system responds with an HTTP `202 Accepted` to the producer.
4. **Processing**: An asynchronous worker pool polls the database for pending deliveries.
5. **Subscription Resolution**: The worker looks up all active HTTP subscriptions associated with the operation name and topic.
6. **Delivery**: The worker builds an HTTP POST request, generates an HMAC signature of the payload, and sends it to the consumer URL.
7. **Resolution**:
    - **Success (2xx)**: The delivery is marked as succeeded.
    - **Failure (Non-2xx or Timeout)**: The delivery is scheduled for retry with exponential backoff.
    - **Max Retries Exceeded**: The message is moved to a Dead Letter Queue (DLQ).
---


## Low level

1. Golang
2. SQLite (sqlc) - `modernc.org/sqlite`
3. Chi router - `go.chi.dev/chi/v5`
4. JSON Schema validation using `github.com/xeipuuv/gojsonschema` to validate JSON payloads on message ingestion.
5. Ingest and Management API Server: HTTP server implementing standard Chi middlewares for logging, recovery, and optional API key verification via header `X-Gosub-Api-Key`.
6. Database Schema & Migrations: Portable SQLite schema using `modernc.org/sqlite` (pure Go driver) and standard SQL migrations stored in `/migrations`, run automatically at startup.
7. SQLC for Type-Safe Database Queries: Auto-generated database queries in `db/generated` from queries defined in `db/query` against the database schema.
8. Background Dispatcher & Worker Pool: Polling-based worker pool that retrieves pending attempts, dispatches HTTP POST webhooks, performs HMAC signature generation, and handles retries with exponential backoff and jitter.
9. Webhook Security and Signing: HMAC-SHA256 signature generation using `crypto/hmac` to sign payloads, sent via `X-Gosub-Signature` header along with `X-Gosub-Timestamp` to prevent replay attacks.
10. Concurrency Semaphores: Webhook dispatch rate-limiting per subscriber using `golang.org/x/sync/semaphore` to protect slow consumers from being overwhelmed.
11. DLQ (Dead Letter Queue): Terminal storage for messages that exhaust their retry attempts, exposing inspection and programmatic replay endpoints.
12. Project Package Layout:
    * `cmd/gosub`: Application entry point (`main.go`).
    * `db/query/`: SQL source queries for SQLC.
    * `db/generated/`: Auto-generated database access code.
    * `migrations/`: Raw SQL migration files.
    * `internal/config/`: Configuration parsing from environment variables.
    * `internal/db/`: SQLite driver setup, database connection pool, and migrations execution.
    * `internal/fixture/`: Deployment contract parser (`fixture.json`) and database synchronizer.
    * `internal/api/`: REST endpoints, routes, middleware, and handlers.
    * `internal/crypto/`: HMAC signing utility.
    * `internal/workers/`: Dispatcher and consumer worker pool.
    * `internal/models/`: Shared models, status constants, and request/response payloads.
