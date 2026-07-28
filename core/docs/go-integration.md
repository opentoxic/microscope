# Go integration

Module: `github.com/opentoxic/microscope/adaptor/go`

The Go adaptor is the reference implementation. It records SQL via `pgx` tracing, HTTP via
middleware, runtime metrics every 15 seconds, and optional Redis/Kafka instrumentation.

## Install

```bash
go get github.com/opentoxic/microscope/adaptor/go
```

Optional scaffolding (config snippets, wire helper):

```bash
go run github.com/opentoxic/microscope/adaptor/go/cmd/install@latest --dir .
```

Flags: `--force`, `--no-env`, `--skip-get`.

### Local development (monorepo)

```go
// go.mod
replace github.com/opentoxic/microscope/adaptor/go => ../path/to/microscope/adaptor/go
```

## Minimal example

Runnable sample: [`adaptor/go/examples/minimal`](../../adaptor/go/examples/minimal/main.go).

```go
import microscope "github.com/opentoxic/microscope/adaptor/go"

ms, err := microscope.Setup(ctx, microscope.SetupOptions{
    AppEnv: os.Getenv("APP_ENV"),
    DSN:    os.Getenv("DATABASE_URL"),
    Logger: log,
})
if err != nil {
    return err
}
defer ms.Close()

mux := http.NewServeMux()
mux.HandleFunc("GET /health", healthHandler)
ms.RegisterRoutes(mux)

handler := ms.HTTPHandler(mux, microscope.HTTPOptions{
    AccessLog: microscope.SimpleAccessLog(ms.Logger),
})
http.ListenAndServe(":8080", handler)
```

Set `APP_ENV=development` and `MICROSCOPE_ENABLED=true`, then open `/microscope`.

## Chi router

When mounting on chi, register routes with **full paths** — do not use `Route("/microscope", …)` which strips the prefix (microscope's `ServeMux` expects `/microscope/...`).

```go
path := strings.TrimRight(ms.Config().Path, "/")
r.Handle(path, microMux)
r.Handle(path+"/*", microMux)
```

See a full wiring example in consumer apps (e.g. tethova-core `internal/microscope/wire.go`).

## Shared database pool

To trace SQL through the same pool your app uses:

```go
ms, _ := microscope.Setup(ctx, opts)
db := stdlib.OpenDBFromPool(ms.Pool) // database/sql over pgx pool
```

Call `Setup` before opening application DB connections when you want a single shared pool.

## HTTP middleware chain

`HTTPHandler` applies:

```
recover → extra (CORS, …) → request ID → bridge → record → access log
```

```go
handler := ms.HTTPHandler(mux, microscope.HTTPOptions{
    Extra: []func(http.Handler) http.Handler{corsMiddleware},
    Bridge: func(ctx context.Context) microscope.RequestMeta {
        return microscope.RequestMeta{RequestID: chimiddleware.GetReqID(ctx)}
    },
    AccessLog: hostAccessLog,
})
```

Use `HTTPMiddlewares` when you need finer control over ordering with an existing stack.

## Optional instrumentation

```go
ms, _ := microscope.Setup(ctx, microscope.SetupOptions{
    AppEnv:  appEnv,
    DSN:     dsn,
    Logger:  log,
    Redis:   redisClient,
    Brokers: []string{"localhost:9092"},
})

publisher = ms.WrapPublishFunc(innerPublish)
notifier = ms.WrapOTPNotifier(innerSendOTP)
```

| Signal | How |
|--------|-----|
| SQL | Automatic via `pgx` query tracer on the setup pool |
| Redis | Pass `Redis` in `SetupOptions` |
| Kafka health | Pass `Brokers`; use `ms.Kafka.Check(ctx)` |
| Kafka I/O | `microscope.NewKafkaWriter` / `NewKafkaReader` |
| Outgoing HTTP | `ms.Hub().WrapHTTPClient(client)` |
| Logs | `log = ms.TeeSlog(log)` after `Setup` |

## Migrations

Embedded migrations run automatically when `MICROSCOPE_AUTO_MIGRATE=true` (default).

Disable:

```go
auto := false
microscope.Setup(ctx, microscope.SetupOptions{AutoMigrate: &auto, /* … */})
```

Manual migration with golang-migrate:

```go
import msmigrate "github.com/opentoxic/microscope/adaptor/go/migrate"

source, _ := msmigrate.Source()
// wire into migrate.NewWithSourceInstance(...)
```

Or call `microscope.MigrateUp(ctx, pool)` directly.

Files (in `core/migrations/`): `001_microscope`, `002_microscope_settings`, `003_microscope_options`.

## Configuration

See [Configuration](configuration.md). Programmatic merge:

```go
cfg := microscope.MergeConfig(microscope.ConfigFromEnv(), overrides)
```

## Shutdown

```go
ms.Close() // closes hub workers and the pgx pool created by Setup
```

## Standalone server

For a separate collector process (legacy sidecar pattern), run `adaptor/go/cmd/server` instead of in-process setup. Prefer in-process integration for new projects.

## See also

- [Getting started](getting-started.md)
- [Architecture](architecture.md)
- [Custom events](tutorials/custom-events.md)
