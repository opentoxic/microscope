# Configuration

All adaptors read the same environment variables. Values below are defaults unless noted.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MICROSCOPE_ENABLED` | `true` | Master switch. Set `false` to disable entirely. |
| `MICROSCOPE_PATH` | `/microscope` | Dashboard and API URL prefix. |
| `MICROSCOPE_RETENTION_HOURS` | `24` | Entries older than this are pruned in the background. |
| `MICROSCOPE_MAX_BODY_BYTES` | `65536` | Maximum request/response body bytes captured per entry. |
| `MICROSCOPE_ALLOWED_ENVS` | `development,local` | Comma-separated `APP_ENV` values where microscope may run. |
| `MICROSCOPE_AUTO_MIGRATE` | `true` | Run embedded SQL migrations on boot when active. |
| `MICROSCOPE_REDACT_SENSITIVE` | `false` | Mask passwords, tokens, and similar fields before storage. |

Laravel also maps these through `config/microscope.php` (published via `php artisan vendor:publish --tag=microscope-config`).

## Activation rules

```
active = MICROSCOPE_ENABLED
     AND APP_ENV ∈ MICROSCOPE_ALLOWED_ENVS
```

When inactive, adaptors skip recording, migrations (if disabled), and route registration overhead is minimal.

## Laravel-specific

| Config key | Env | Notes |
|------------|-----|-------|
| `microscope.enabled` | `MICROSCOPE_ENABLED` | Defaults to `false` in published config — set `true` in `.env`. |
| `microscope.path` | `MICROSCOPE_PATH` | Route prefix without leading slash (e.g. `microscope`). |
| `microscope.middleware` | — | Default `['web']`. Add auth middleware for non-local environments. |
| `microscope.max_body_bytes` | `MICROSCOPE_MAX_BODY_BYTES` | Passed to request capture middleware. |
| `microscope.redact_sensitive` | `MICROSCOPE_REDACT_SENSITIVE` | Boolean. |

## Go `SetupOptions`

Programmatic overrides merge on top of `ConfigFromEnv()`:

```go
ms, err := microscope.Setup(ctx, microscope.SetupOptions{
    AppEnv: "development",
    DSN:    os.Getenv("DATABASE_URL"),
    Config: microscope.Config{Path: "/debug"},
    Redis:  redisClient,           // optional Redis hook
    Brokers: []string{"localhost:9092"}, // optional Kafka health
})
```

Only set `Config` fields you intend to override. An empty `Config{}` does not disable `MICROSCOPE_ENABLED` from the environment.

## Security notes

- Keep microscope **disabled in production** unless you explicitly allow the env and protect routes with authentication.
- Use `MICROSCOPE_REDACT_SENSITIVE=true` when capturing real credentials in shared environments.
- In Laravel, restrict `/microscope` with middleware (e.g. super-admin gate) outside `local`/`development`.

## Dashboard settings

Some policies are also editable live under **Settings** in the UI (signal types, recording pause, redaction). These persist in the `microscope_settings` and `microscope_options` tables.
