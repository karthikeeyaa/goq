# goq System Knowledge

This document describes the implementation that exists in the repository. It is not a design proposal. The main entrypoint is `cmd/main.go`; package responsibilities follow the directory layout under `internal/`.

## Runtime Model

goq is a single-process, single-node HTTP service. SQLite is the control plane for topic metadata. The message plane is a pair of local files per topic. There is no replication, consumer group coordinator, background dispatcher, archive store, or runtime topic-management API.

The process owns one `LogFile` for every topic returned by `ListTopics`. Each `LogFile` contains:

- `Topic`: the sqlc-generated `store.Topic` configuration.
- `File`: the topic message file.
- `IndexFile`: the topic offset index.
- `mu`: a `sync.Mutex` shared by append, read, and close operations for that topic.

Different topics can be served independently. Operations for the same topic are serialized by its mutex.

## Startup and Shutdown

`cmd/main.go` creates a context that is cancelled by `SIGINT` or `SIGTERM`, then performs this sequence:

1. `config.LoadConfig()` loads `.env` if present and reads environment variables.
2. `logger.Init()` creates the configured log directory and opens an append-only log file. Logs go to both stdout and the file.
3. `db.Connect()` creates the SQLite parent directory and database file, opens SQLite, pings it, enables foreign keys, and sets `PRAGMA journal_mode=DELETE`.
4. `db.RunMigrations()` reads embedded SQL files from `migrations/`, sorts them by filename, and applies missing migrations in one transaction.
5. Each applied migration is recorded in `goq_migrations` with a SHA-256 hash. A changed file with the same name causes startup to fail.
6. `fixture.CreateFixtures()` reads `build/fixture.json` and upserts its topics in SQLite.
7. `logstore.Init()` lists topics and opens their message and index files.
8. `api.StartServer()` starts the HTTP server and waits for either a server error or context cancellation.

On shutdown, the HTTP server is given a timeout-based graceful shutdown and the database, logger, and log files are closed through deferred calls. The retention worker is currently an empty package and is not started.

## Control Plane

Migration `migrations/001_migration.schema.sql` creates:

```sql
goq_migrations(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		hash TEXT NOT NULL,
		applied DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
)
```

Migration `migrations/002_topic.schema.sql` creates `topic`:

```sql
topic(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		retention_ms INTEGER DEFAULT 86400000,
		cleanup_policy TEXT DEFAULT 'delete'
				CHECK (cleanup_policy IN ('delete', 'compact')),
		max_message_bytes INTEGER DEFAULT 1048576,
		log_index_interval_bytes INTEGER DEFAULT 4096,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL
)
```

`db/query/topic.sql` currently exposes only:

- `GetTopic(name)`: fetch one topic for request validation.
- `ListTopics()`: load all topics during log-store initialization.
- `UpsertTopic(...)`: insert or update fixture configuration.

The generated `store.Topic` uses `sql.NullInt64` and `sql.NullString` for the optional configuration values. Topic rows are not dynamically reflected into an already-running `LogStore`; the log store is built once during startup.

## Configuration

`internal/config/config.go` currently loads these values:

| Variable | Default | Use |
| --- | --- | --- |
| `APP_NAME` | `goq` | Application name in configuration |
| `PORT` | `8080` | HTTP listen port |
| `VERSION` | `v1` | API version path component |
| `MODE` | `production` | Disables auth when set to `development` |
| `LOGS_DIRECTORY` | `logs` | Parent directory for the application log |
| `HTTP_TIMEOUT` | `30` | Request and shutdown timeout in seconds |
| `INTEGRATION_KEY` | empty | Optional API authentication key |
| `DB_DSN` | `data/goq.db` | SQLite database path/DSN |
| `MESSAGES_DIR` | `data/messages` | Directory containing message files |
| `FIXTURE_FILE` | `build/fixture.json` | Topic fixture path |

`Config` also declares `LogSegmentBytes`, `LogIndexIntervalBytes`, and `MaxMessageBytes`, but `LoadConfig()` does not populate them. The active per-topic values come from SQLite and the fixture instead.

## Filesystem Message Store

The current layout is flat, not segmented by directory:

```text
MESSAGES_DIR/
	orders.log
	orders.index
	notifications.log
	notifications.index
```

The topic name is directly used in the filename. Topic-name validation is currently limited to checking that the URL parameter is non-empty; path-safety validation is not performed by `logstore.Init()`.

### Log Record Format

Each append writes one variable-length record. All integer fields use little-endian encoding:

| Bytes | Field |
| ---: | --- |
| 4 | CRC-32/IEEE of the payload |
| 4 | Payload length as `uint32` |
| 8 | Unix timestamp in milliseconds as `uint64` |
| N | JSON-marshaled payload bytes |

The record header is therefore 16 bytes and the total record size is `16 + N` bytes. There is no magic byte and the record does not store its logical message offset.

### Index Format

Every published message appends exactly one 12-byte index entry:

| Bytes | Field |
| ---: | --- |
| 8 | Absolute byte position of the record in the `.log` file as `uint64` |
| 4 | Payload length as `uint32` |

The logical message offset is calculated as `index file size / 12` before the new entry is written. Offset zero is the first index entry. Reading offset `T` seeks to `T * 12` in the index, so the current index is dense rather than sparse.

## Append Path

`internal/api/producer.go` first obtains the topic from request context, looks up its already-open `LogFile`, and decodes the request body as:

```json
{"payload": <any valid JSON value>}
```

An absent or empty `payload` returns HTTP 400. `json.RawMessage` preserves the JSON bytes received for the payload value. `LogFile.Append()` then:

1. Locks the topic mutex.
2. Marshals the payload again with `json.Marshal`.
3. Rejects a zero-length payload or one larger than `Topic.MaxMessageBytes` when that SQL value is valid.
4. Seeks the log file end and records the byte position.
5. Creates the millisecond timestamp and CRC-32 checksum.
6. Writes the 16-byte header and payload to the log file.
7. Computes the next logical offset from the current index file size.
8. Writes the 12-byte index entry.
9. Seeks both file handles back to their ends.

The response is HTTP 202 and contains `status`, `topic`, `offset`, `max_message_bytes_limit`, and `timestamp_ms`. The implementation does not call `Sync()`, does not use a transaction across the two files, and does not explicitly handle partial writes. A failure between the log write and index write can leave the files inconsistent.

## Pull Path

`internal/api/consumer.go` accepts:

- `offset`: optional non-negative integer, default `0`.
- `limit`: optional positive integer, default `100`, capped at `1000`.

Malformed or negative offsets and invalid limits return HTTP 400. `LogFile.Read(offset, limit)` locks the topic mutex, calculates the high watermark as `index file size / 12`, and returns no messages when the requested offset is at or beyond that watermark.

For each requested index entry, it reads the absolute log position and payload length, seeks to the record, reads the 16-byte header and payload, and returns:

```json
{
	"offset": 0,
	"timestamp_ms": 1700000000000,
	"payload": {}
}
```

The pull response includes `status`, `count`, `topic`, `requested_offset`, `next_offset`, `high_watermark`, and `results`. `next_offset` is calculated as `requested_offset + number of messages returned`.

Although the record stores a CRC, the read path currently does not compare it with a newly calculated checksum. It also trusts the index and payload length; truncated or corrupt files return an internal error rather than being recovered or repaired. The API uses Go `int` for offsets and watermarks.

## HTTP Routing and Middleware

The router is created in `internal/api/server.go`:

- `GET /admin/health` returns `{"status":"ok"}`.
- `POST /api/{VERSION}/push/{topic}/` invokes `Push`.
- `GET /api/{VERSION}/pull/{topic}/` invokes `Pull`.

The API router uses `AuthMiddleware`. Authentication is bypassed when `MODE=development` or `INTEGRATION_KEY` is empty. Otherwise the request must contain exactly:

```text
Authorization: token <INTEGRATION_KEY>
```

Global middleware adds a request ID, derives the client IP, logs the request, recovers panics, and applies the configured timeout. `ValidateTopic` queries SQLite for every publish and pull request. Unknown topics return HTTP 404. A missing log file for a known topic returns HTTP 404 with `LOG_FILE_NOT_FOUND`.

Errors use this shape:

```json
{
	"status": "error",
	"code": "INVALID_INPUT",
	"message": "..."
}
```

The router's custom not-found handler attempts to hijack and close the connection, then falls back to a plain 404 response rather than the normal JSON error shape.

## Fixtures

`build/fixture.json` currently defines `orders` and `notifications`. Fixture fields are:

```json
{
	"name": "orders",
	"retention_ms": 604800000,
	"cleanup_policy": "delete",
	"max_message_bytes": 2097152,
	"log_index_interval_bytes": 4096
}
```

The fixture loader validates non-empty names and allows only `delete` or `compact` cleanup policies. It upserts topics in SQLite. `compact` is accepted and stored but no compaction behavior exists. A missing or invalid fixture file stops startup.

## Current Limitations

- No startup scan, offset recovery, truncation, or repair of existing log/index files.
- No log segments; `LogSegmentBytes` is not active.
- No sparse index; every message gets an index entry.
- CRC is written but not validated when reading.
- No `fsync`, atomic append protocol, or crash-consistency guarantee.
- No retention implementation; `internal/workers/retention.go` only declares the package.
- No binary payload API; publish requires JSON and stores JSON bytes.
- No runtime topic create/update/delete endpoints.
- No metrics, readiness status, rate limiting, request body limit, or disk-space monitoring.
- No replication, archive storage, push delivery, retries, or dead-letter queue.

## Development Commands

```bash
make deps       # run go mod tidy
make build      # format and build build/goq
make run        # run cmd/main.go
make sqlc       # regenerate db/store from sqlc.yaml
make fmt        # format Go files
```

`Makefile` declares `test` and `clean` as phony targets, but currently provides no recipes for them. The repository should be tested with direct Go tooling until those targets are implemented.