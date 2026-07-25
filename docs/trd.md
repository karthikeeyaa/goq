# Technical Specification: goq

A lightweight message log for internal microservice integration. Domain-agnostic, plug-and-play. Append-only topic logs, offset-based pull, sparse file indexes.

## Storage Architecture

| Plane | Technology | Holds |
| --- | --- | --- |
| **Message plane** | Files (`.log` + `.index`) | Payloads, offsets, hot path produce/consume |
| **Control plane** | SQLite + **sqlc** | Topic registry |

## Requirements

* Producers publish opaque binary/JSON payloads to a **topic**.
* Consumers **pull** by offset (`from` + `limit`).
* Each topic is one independent **append-only log**.
* On disk per topic: **log segment(s) + sparse offset index**.
* Retention deletes old segments in place.
* Consumers track offset client-side.
* If `from` is omitted on pull, return empty messages + current **head offset**.
* Topics live in SQLite (`topics` table). Unknown topic → 404.
* Single writer path per topic (serialized append + monotonic offset).

## Control Plane (SQLite + sqlc)

SQLite acts as the metadata registry. Migrations run on startup.

```sql
CREATE TABLE topic (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    retention_ms INTEGER DEFAULT 86400000,
    cleanup_policy TEXT DEFAULT 'delete' CHECK (cleanup_policy IN ('delete', 'compact')),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Startup Flow:**
1. Connect SQLite (`DB_DSN`), run migrations.
2. `SELECT` all topics via sqlc.
3. For each name: ensure `{DATA_DIR}/{name}/` exists; open/create log + index; recover end offset.
4. Build in-memory map `name → TopicLog` for the API.

## Message Plane (Filesystem)

```text
{DATA_DIR}/                    
  payments/
    00000000000000000000.log
    00000000000000000000.index
```

### Record Format (Binary)
`| magic 1 | length 4 | offset 8 | timestamp_ms 8 | crc32 4 | payload N bytes |`

### Sparse Offset Index (`.index`)
| Entry | Size |
| --- | ---: |
| offset | 8 |
| file position | 8 |

* Appended every `LOG_INDEX_INTERVAL_BYTES` (default 4096).
* Lookup `T`: binary search largest entry ≤ `T` → seek log → scan forward.

## HTTP API

| Method | Path | Behavior |
| --- | --- | --- |
| `GET` | `/health` | Liveness |
| `POST` | `/api/v1/publish/{topic}` | Body = payload; `202 Accepted` + `{ "topic", "offset" }` |
| `GET` | `/api/v1/pull/{topic}?from=&limit=` | Fetch after offset |

* Errors: unknown topic → 404; bad `from` → 400; empty body → 400. 
* Auth: optional `INTEGRATION_KEY` header.

## Stack & Packages

1. **Go**, **Chi** HTTP router
2. **SQLite** (`modernc.org/sqlite`) + **sqlc** + migrations
3. **Filesystem logstore**

```text
internal/
  config/       # env variables
  db/           # connect, migrate, sqlc queries
  logstore/     # TopicLog: Append, ReadFrom, Head, recover
  api/          # publish / pull / health
  workers/      # retention ticker
```

## Phase 2 (Future Capabilities)
* **Push Delivery**: Background dispatcher to push messages to HTTP subscriptions (webhooks), with configurable retry backoff and Dead Letter Queues (DLQ).
* **Archive Storage**: Automatically move older, expired messages out of the active log and into archive storage, with support for transparent historical replay from the archive.
