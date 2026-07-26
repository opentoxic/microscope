# microscope — development observability, any stack

Records HTTP requests, SQL queries, structured logs, cache and Redis activity, queue jobs,
schedules, events, notifications, mail, outgoing HTTP calls, WebSockets, performance spans,
runtime metrics, custom events, and panics during development. Ships a Vue 3 workspace served
by the Go core.

## Layout

- **Go core** (repo root, module `github.com/qobly/microscope`): the hub, storage, HTTP API,
  and embedded UI. Import it directly in a Go service, or run it standalone via `cmd/server`.
- **`sdk/python`**: pip package (`microscope-client`), a thin HTTP client for the same API.
- **`sdk/typescript`**: npm package (`@qobly/microscope-client`), same idea for Node/browser apps.

Any stack can record entries by calling the HTTP API directly, so the Go core doubles as the
one service every language's SDK talks to.

## Requirements

- Go 1.25+
- PostgreSQL (for entry storage)
- Node.js 20+ (to build the UI)

## Quick start (standalone server)

1. Run migrations `migrations/001_microscope.up.sql` and `migrations/002_microscope_settings.up.sql`
2. Set `APP_ENV=development` and `MICROSCOPE_ENABLED=true`
3. `make run` — builds the Vue UI and starts the server
4. Open `/microscope`

## Using the Go module directly (in-process)

```bash
go get github.com/qobly/microscope
```

```go
cfg := microscope.DefaultConfig()
cfg.Enabled = true
cfg.Path = "/microscope"

hub := microscope.NewWithTracer(pool, cfg, logger, microscope.NewQueryTracer())

// Optional: tee logs to microscope
logger = slog.New(microscope.NewSlogHandler(logger.Handler(), hub))
```

### Bridge request metadata

```go
microscope.BridgeMiddleware(func(ctx context.Context) microscope.RequestMeta {
    return microscope.RequestMeta{RequestID: "...", IPAddress: "..."}
})
```

### Register HTTP routes and middleware

```go
(&microscope.Handler{Hub: hub}).RegisterRoutes(mux)

chain := []func(http.Handler) http.Handler{
    yourRequestIDMiddleware,
    microscope.BridgeMiddleware(...),
    hub.Middleware(),
    hub.RecoverMiddleware(logger),
}
```

### Optional watchers

- **SQL**: pass `hub.QueryTracer()` to your pgx pool as `QueryTracer`
- **Redis**: install `microscope.NewRedisHook(hub)` with `redisClient.AddHook`
- **Kafka / Redpanda**: wrap kafka-go clients with `microscope.NewKafkaWriter(writer, hub)` and `microscope.NewKafkaReader(reader, hub)`
- **Outgoing HTTP**: use `hub.WrapHTTPClient(client)`
- **Cache, jobs, schedules, mail, WebSockets, metrics, and custom events**: use the typed `hub.Record*` methods
- **Performance spans**: `done := hub.Timed(ctx, "operation", fields)` and call `done(err)` when the operation finishes

New entries are pushed to the workspace over server-sent events. Polling is retained only as a
low-frequency recovery path.

## Using microscope from other stacks

Every stack talks to the same HTTP API, so a service written in Python, Node, or anything else
can record entries without touching Go:

```bash
pip install microscope-client       # Python
npm install @qobly/microscope-client  # TypeScript / JavaScript
```

```python
from microscope_client import MicroscopeClient

client = MicroscopeClient(base_url="http://localhost:8093/microscope")
client.record("payment_charged", tags=["billing"], content={"amount": 4200})
```

```ts
import { MicroscopeClient } from "@qobly/microscope-client";

const client = new MicroscopeClient({ baseUrl: "http://localhost:8093/microscope" });
await client.record("payment_charged", { tags: ["billing"], content: { amount: 4200 } });
```

See `sdk/python/README.md` and `sdk/typescript/README.md` for full SDK docs.

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/microscope/api/entries` | List entries (`type`, `search`, `limit`, `offset`) |
| GET | `/microscope/api/entries/{id}` | Entry detail with batch groups and content tabs |
| GET | `/microscope/api/stream` | Live server-sent entry stream |
| POST | `/microscope/api/entries` | Record a named custom timeline marker |
| POST | `/microscope/api/prune` | Clear all entries |
| GET | `/microscope/api/settings` | List recording policies and retained counts |
| PUT | `/microscope/api/settings/{type}` | Enable or disable a signal (`{"enabled":false}`) |

Disabling a signal is destructive by design: the hub closes its ingestion gate, transactionally
deletes every retained entry of that type, and broadcasts the mutation to connected dashboards.
Re-enabling it resumes recording new entries; deleted history is not restored.

## UI development

```bash
cd ui
npm run dev   # Vite on :5173, proxies API to :8093
```

## Configuration

| Field | Default | Description |
|-------|---------|-------------|
| `Enabled` | `true` | Feature flag (also gated by `Enabled(env, cfg)`) |
| `Path` | `/microscope` | Dashboard and API prefix |
| `RetentionHours` | `24` | Auto-prune entries older than this |
| `MaxBodyBytes` | `65536` | Max request/response body captured |

microscope is active only when `APP_ENV=development` and `Enabled` is true.
