# microscope — development observability, any stack

Records HTTP requests, SQL queries, structured logs, cache and Redis activity, queue jobs,
schedules, events, notifications, mail, outgoing HTTP calls, WebSockets, performance spans,
runtime metrics, custom events, and panics during development. Ships a Vue 3 workspace served
by the Go core.

## Layout

- **Go core** (repo root, module `github.com/opentoxic/microscope`): the hub, storage, HTTP API,
  and embedded UI. Import it directly in a Go service, or run it standalone via `cmd/server`.
- **`sdk/python`**: pip package (`microscope-client`), with `Django` and `FastAPI` integrations.
- **`sdk/typescript`**: npm package (`@opentoxic/microscope-client`), with `Express` and `NestJS` integrations.
- **`sdk/ruby`**: gem (`microscope_client`), stdlib-only.
- **`sdk/php`**: composer package (`opentoxic/microscope-client`), with a `Laravel` service provider.
- **`sdk/elixir`**: hex package (`microscope_client`).

Any stack can record entries by calling the HTTP API directly, so the Go core doubles as the
one service every language's SDK talks to. Each SDK is a thin wrapper around that API, so
adding another language means writing a small HTTP client, not re-implementing the service.

## Requirements

- Go 1.25+
- PostgreSQL (for entry storage)
- Node.js 20+ (to build the UI)

## Quick start (standalone server)

1. Run migrations `migrations/001_microscope.up.sql`, `002_microscope_settings.up.sql`, and `003_microscope_options.up.sql`
2. Set `APP_ENV=development` and `MICROSCOPE_ENABLED=true`
3. `make run` — builds the Vue UI and starts the server
4. Open `/microscope`

## Zero-setup interactive UI demo

To explore the complete dashboard without PostgreSQL or a running collector:

```bash
cd ui
npm install
npm run demo
```

Open `http://127.0.0.1:5173/microscope/`. Demo mode includes a live polyglot
signal stream, linked traces, SQL and JSON inspection, interactive topology
diagrams, bookmarks, filters, and all appearance settings. It never sends demo
data to a backend.

## Using microscope in Go services

Microscope runs **in-process** inside your Go service. It shares your PostgreSQL database, records signals as your app handles traffic, and serves the Vue dashboard on a path such as `/microscope`.

For advanced topics (golang-migrate, manual pool wiring, middleware ordering), see **[docs/go-integration.md](docs/go-integration.md)**. A runnable example lives in **[examples/minimal](examples/minimal/main.go)**.

### Prerequisites

- Go 1.25+
- PostgreSQL (same database as your app, or a dedicated dev database)
- `APP_ENV` set to `development` or `local` (configurable)

### 1. Add the dependency

```bash
go get github.com/opentoxic/microscope
```

Optional scaffolding for config snippets and wire helpers:

```bash
go run github.com/opentoxic/microscope/cmd/install@latest --dir .
```

### 2. Call `Setup` at startup

`Setup` creates the pgx pool, runs migrations when active, binds the hub, and tees structured logs into microscope.

```go
ms, err := microscope.Setup(ctx, microscope.SetupOptions{
    AppEnv: os.Getenv("APP_ENV"),
    DSN:    os.Getenv("DATABASE_URL"),
    Logger: log,
    Config: microscope.Config{
        Path:            "/microscope",
        RedactSensitive: false, // full payloads in local dev; set true to mask secrets
    },
})
if err != nil {
    return err
}
defer ms.Close()
```

When `APP_ENV` is outside `AllowedEnvs` or `Enabled` is false, `ms.Active()` is false and microscope becomes a no-op — safe to keep the wiring in production builds.

Pass optional integrations in the same call:

```go
ms, err := microscope.Setup(ctx, microscope.SetupOptions{
    AppEnv:  appEnv,
    DSN:     dsn,
    Logger:  log,
    Redis:   redisClient,              // attaches Redis hook automatically
    Brokers: []string{"localhost:9092"}, // enables Kafka health probes
    PoolConfig: func(cfg *pgxpool.Config) {
        cfg.MaxConns = 25
    },
})
```

Reuse `ms.Pool` for your repositories so SQL queries are traced automatically.

### 3. Mount routes and wrap your HTTP handler

Register your application routes first, then mount microscope, then wrap the mux with `HTTPHandler`:

```go
mux := http.NewServeMux()
mux.HandleFunc("GET /health", healthHandler)
// ... your API routes ...

if ms.Active() {
    ms.RegisterRoutes(mux) // dashboard + API under /microscope
}

handler := ms.HTTPHandler(mux, microscope.HTTPOptions{
    AccessLog: microscope.SimpleAccessLog(ms.Logger),
    Extra: []func(http.Handler) http.Handler{
        corsMiddleware,
    },
    Bridge: func(ctx context.Context) microscope.RequestMeta {
        return microscope.RequestMeta{
            RequestID:     requestIDFromContext(ctx),
            CorrelationID: correlationIDFromContext(ctx),
        }
    },
})

http.ListenAndServe(":8080", handler)
```

`HTTPHandler` builds this chain:

```
recover → extra (CORS, etc.) → request ID → bridge → record → access log
```

Incoming HTTP requests, panics, and response bodies are recorded automatically. Access logs skip `/microscope` paths by default.

If you already have a middleware stack, use `HTTPMiddlewares` and compose it yourself — see [docs/go-integration.md](docs/go-integration.md).

### 4. Set environment variables

```bash
APP_ENV=development
MICROSCOPE_ENABLED=true
DATABASE_URL=postgres://user:pass@localhost:5432/myapp?sslmode=disable

# optional
MICROSCOPE_PATH=/microscope
MICROSCOPE_RETENTION_HOURS=24
MICROSCOPE_MAX_BODY_BYTES=65536
MICROSCOPE_AUTO_MIGRATE=true
MICROSCOPE_REDACT_SENSITIVE=false
```

### 5. Run and open the dashboard

```bash
go run .
```

Open `http://localhost:8080/microscope`. New entries stream live over server-sent events.

### 6. Instrument optional dependencies

| Signal | How to wire |
|--------|-------------|
| **SQL** | Automatic when you use `ms.Pool` from `Setup` |
| **Redis** | Pass `Redis` in `SetupOptions`, or `redisClient.AddHook(ms.RedisHook())` |
| **Kafka / Redpanda** | `microscope.NewKafkaWriter(w, ms.Hub())` / `NewKafkaReader(r, ms.Hub())` |
| **Outgoing HTTP** | `client = ms.Hub().WrapHTTPClient(client)` |
| **Structured logs** | Use `ms.Logger` returned from `Setup` (tees into microscope) |
| **Domain events** | `ms.WrapEventPublisher(inner)` or `microscope.RecordPublish(ms.Hub(), ctx, type, payload)` |
| **OTP / mail** | `ms.WrapOTPNotifier(inner)` or `microscope.RecordOTP(ms.Hub(), ctx, kind, email, otp)` |

Kafka producer example:

```go
writer := &kafka.Writer{Addr: kafka.TCP("localhost:9092"), Topic: "events"}
wrapped := microscope.NewKafkaWriter(writer, ms.Hub())
_ = wrapped.WriteMessages(ctx, kafka.Message{Value: []byte(`{"ok":true}`)})
```

Wrap an existing publisher or notifier without changing call sites:

```go
publisher = ms.WrapPublishFunc(innerPublish)
sendOTP   = ms.WrapOTPFunc("signup_otp", innerSendOTP)
```

### 7. Record custom signals

Use the hub for anything not covered by automatic instrumentation:

```go
hub := ms.Hub()

// cache, jobs, schedules, mail, websockets, metrics
hub.RecordCache(ctx, "get", "user:42", true, time.Millisecond*2, nil)
hub.RecordJob(ctx, "send-receipt", "default", "done", time.Second, nil)

// named custom timeline markers
hub.RecordCustom(ctx, "payment_captured", map[string]any{
    "amount_cents": 4200,
    "currency":     "USD",
})

// performance spans — call the returned func when the operation finishes
done := hub.Timed(ctx, "stripe.charge", map[string]any{"customer": "cus_123"})
err := chargeCustomer()
done(err)
```

POST to the API from anywhere (including non-Go services) with the same effect:

```bash
curl -X POST http://localhost:8080/microscope/api/entries \
  -H 'Content-Type: application/json' \
  -d '{"name":"deploy_started","content":{"version":"1.2.3"}}'
```

### 8. Sensitive data and redaction

By default, microscope stores **full dev payloads** — HTTP bodies, SQL args, Redis commands, Kafka messages, and OTPs — so you can debug locally.

Enable redaction when payloads should be masked before storage:

```go
Config: microscope.Config{RedactSensitive: true}
```

Or at runtime via env / dashboard:

```bash
MICROSCOPE_REDACT_SENSITIVE=true
```

```bash
curl -X PUT http://localhost:8080/microscope/api/redaction \
  -H 'Content-Type: application/json' \
  -d '{"enabled":true}'
```

### 9. Manual wiring (existing pgx pool)

When you manage the pool yourself instead of using `Setup`:

```go
integ := microscope.NewIntegration(appEnv, cfg)
poolCfg, _ := pgxpool.ParseConfig(dsn)
integ.ConfigurePool(poolCfg) // attach query tracer before pool creation
pool, _ := pgxpool.NewWithConfig(ctx, poolCfg)

if integ.Active() {
    integ.Bind(pool, log)
    log = integ.TeeSlog(log)
    redisClient.AddHook(integ.RedisHook())
}
```

### What gets recorded automatically

With the HTTP handler and shared pool wired, each request produces a linked trace: HTTP span, SQL queries, logs, Redis/Kafka activity, outgoing HTTP calls, panics, and any custom signals you add — grouped by request/batch context in the dashboard.

## Using microscope from other stacks

Every stack talks to the same HTTP API, so a service written in Python, Node, or anything else
can record entries without touching Go:

```bash
pip install microscope-client          # Python (Django, FastAPI integrations included)
npm install @opentoxic/microscope-client   # TypeScript / JavaScript (Express, NestJS integrations included)
gem install microscope_client          # Ruby
composer require opentoxic/microscope-client  # PHP (Laravel integration included)
mix deps.get microscope_client         # Elixir
```

```python
from microscope_client import MicroscopeClient

client = MicroscopeClient(base_url="http://localhost:8093/microscope")
client.record("payment_charged", content={"amount": 4200})
```

```ts
import { MicroscopeClient } from "@opentoxic/microscope-client";

const client = new MicroscopeClient({ baseUrl: "http://localhost:8093/microscope" });
await client.record("payment_charged", { amount: 4200 });
```

```ruby
client = MicroscopeClient::Client.new(base_url: "http://localhost:8093/microscope")
client.record("payment_charged", content: { amount: 4200 })
```

```php
$client = new Opentoxic\Microscope\MicroscopeClient("http://localhost:8093/microscope");
$client->record("payment_charged", ["amount" => 4200]);
```

```elixir
client = MicroscopeClient.new("http://localhost:8093/microscope")
MicroscopeClient.record(client, "payment_charged", %{amount: 4200})
```

See each SDK's README for full docs and framework integrations:
[`sdk/python`](sdk/python/README.md) (Django, FastAPI) ·
[`sdk/typescript`](sdk/typescript/README.md) (Express, NestJS) ·
[`sdk/ruby`](sdk/ruby/README.md) ·
[`sdk/php`](sdk/php/README.md) (Laravel) ·
[`sdk/elixir`](sdk/elixir/README.md)

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/microscope/api/entries` | List entries (`type`, `search`, `limit`, `offset`) |
| GET | `/microscope/api/entries/{id}` | Entry detail with batch groups and content tabs |
| GET | `/microscope/api/stream` | Live server-sent entry stream |
| POST | `/microscope/api/entries` | Record a named custom timeline marker |
| POST | `/microscope/api/prune` | Clear all entries |
| GET | `/microscope/api/settings` | List recording policies and retained counts |
| PUT | `/microscope/api/settings/{type}` | Enable or disable a recorder (`{"enabled":false}`) |
| GET | `/microscope/api/redaction` | Read sensitive-data redaction policy |
| PUT | `/microscope/api/redaction` | Enable or disable redaction (`{"enabled":true}`) |

Disabling a recorder is destructive by design: the hub closes its ingestion gate, transactionally
deletes every retained entry of that type, and broadcasts the mutation to connected dashboards.
Re-enabling it resumes recording new entries; deleted history is not restored.

By default, Microscope records full payloads for local debugging (passwords, tokens, OTPs, Redis args, Kafka messages, HTTP bodies). Enable redaction in Settings → Recording or with `MICROSCOPE_REDACT_SENSITIVE=true` to mask sensitive fields before storage.

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
| `AllowedEnvs` | `development`, `local` | App environments where microscope runs |
| `AutoMigrate` | `true` | Run embedded migrations on `Setup()` |
| `RedactSensitive` | `false` | Mask passwords, tokens, OTPs, and broker payloads before storage |

microscope is active when `APP_ENV` is in `AllowedEnvs` and `Enabled` is true.
