# Go in-process integration

Import `github.com/opentoxic/microscope` when your service shares a PostgreSQL database with microscope storage.

## Quick start

```bash
go get github.com/opentoxic/microscope
go run github.com/opentoxic/microscope/cmd/install@latest   # optional scaffolding
```

```go
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

http.ListenAndServe(":8080", ms.HTTPHandler(mux, microscope.HTTPOptions{
    AccessLog: microscope.SimpleAccessLog(ms.Logger),
}))
```

Set `APP_ENV=development` (or `local`) and `MICROSCOPE_ENABLED=true`. Open `/microscope`.

See [`examples/minimal`](../examples/minimal/main.go) for a runnable app.

## Requirements

- Go 1.25+
- PostgreSQL
- `APP_ENV` in `development` or `local` (configurable via `AllowedEnvs`)

## Install CLI

Scaffolds config, env snippets, and a wire helper:

```bash
go run github.com/opentoxic/microscope/cmd/install@latest --dir .
```

Flags: `--force` overwrite files, `--no-env` skip `.env.example`, `--skip-get` skip `go get`.

## Migrations

By default `Setup` runs embedded migrations when active (`Config.AutoMigrate`, overridable with `MICROSCOPE_AUTO_MIGRATE`).

### Auto-migrate (recommended)

Enabled by default in `DefaultConfig()`. Disable explicitly:

```go
auto := false
microscope.Setup(ctx, microscope.SetupOptions{
    AutoMigrate: &auto,
})
```

### golang-migrate pipeline

```go
import (
    "github.com/golang-migrate/migrate/v4/source/iofs"
    msmigrate "github.com/opentoxic/microscope/migrate"
)

source, err := msmigrate.Source()
```

Or call `microscope.MigrateUp(ctx, pool)` directly.

Migration files: `001_microscope.up.sql`, `002_microscope_settings.up.sql`, `003_microscope_options.up.sql`.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MICROSCOPE_ENABLED` | `true` | Master switch |
| `MICROSCOPE_PATH` | `/microscope` | Dashboard mount path |
| `MICROSCOPE_RETENTION_HOURS` | `24` | Entry retention |
| `MICROSCOPE_MAX_BODY_BYTES` | `65536` | Max captured request body |
| `MICROSCOPE_ALLOWED_ENVS` | `development,local` | Comma-separated app envs |
| `MICROSCOPE_AUTO_MIGRATE` | `true` | Run migrations on Setup |
| `MICROSCOPE_REDACT_SENSITIVE` | `false` | Mask sensitive fields before storage |

By default Microscope stores full dev payloads. Set `MICROSCOPE_REDACT_SENSITIVE=true` or use Settings → Recording → **Redact sensitive data** to restore masked capture.

YAML config can be merged:

```go
cfg := microscope.MergeConfig(microscope.ConfigFromEnv(), yamlOverrides)
```

## HTTP middleware

`HTTPHandler` builds the full chain:

```
recover → extra (CORS, etc.) → request ID → bridge → record → access log
```

```go
handler := ms.HTTPHandler(mux, microscope.HTTPOptions{
    Extra: []func(http.Handler) http.Handler{corsMiddleware},
    AccessLog: hostAccessLog,
    Bridge: func(ctx context.Context) microscope.RequestMeta {
        return microscope.RequestMeta{RequestID: hostRequestID(ctx)}
    },
})
```

Use `HTTPMiddlewares` instead when you need finer control over ordering with an existing middleware stack.

## Optional instrumentation

```go
ms, _ := microscope.Setup(ctx, microscope.SetupOptions{
    AppEnv:  appEnv,
    DSN:     dsn,
    Logger:  log,
    Redis:   redisClient,
    Brokers: []string{"localhost:9092"},
    PoolConfig: func(cfg *pgxpool.Config) {
        cfg.MaxConns = 25
    },
})

publisher = ms.WrapPublishFunc(innerPublish)
notifier = ms.WrapOTPFunc("signup_otp", innerSendOTP)
```

- **SQL**: tracer attached automatically via `Setup` / `ConfigurePool`
- **Redis**: pass `Redis` in `SetupOptions`
- **Kafka**: pass `Brokers`; use `ms.Kafka.Check(ctx)` for health probes
- **Kafka producers/consumers**: `microscope.NewKafkaWriter` / `NewKafkaReader`
- **Outgoing HTTP**: `ms.Hub().WrapHTTPClient(client)`

## Advanced manual wiring

When you manage the pgx pool yourself:

```go
ms := microscope.NewIntegration(appEnv, cfg)
poolCfg, _ := pgxpool.ParseConfig(dsn)
ms.ConfigurePool(poolCfg)
pool, _ := pgxpool.NewWithConfig(ctx, poolCfg)

if ms.Active() {
    ms.Bind(pool, log)
    log = ms.TeeSlog(log)
}
```

## Shutdown

```go
ms.Close() // or SetupResult.Close() which also closes the pool
```

## Standalone server

Run `cmd/server` instead of in-process integration when you prefer a separate collector service and language SDKs.
