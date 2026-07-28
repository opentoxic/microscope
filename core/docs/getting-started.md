# Getting started

microscope records HTTP traffic, SQL, logs, Redis, queues, runtime metrics, and more during
development. Each **language adaptor** embeds the dashboard and API in your app process and
stores entries in your existing PostgreSQL database.

## Prerequisites

- **PostgreSQL** — entry storage (same database as your app)
- **Node.js 20+** and **pnpm** — only if you build the dashboard from source
- Your stack's runtime: Go 1.25+, PHP 8.1+, or Python 3.10+

## Option 1 — UI demo (no backend)

Explore the dashboard with mock data:

```bash
cd core/ui
pnpm install
pnpm run demo
```

Open `http://127.0.0.1:5173/microscope/`. Nothing is written to a database.

## Option 2 — Standalone Go server

Run the reference server from this repository:

```bash
# from repo root
export APP_ENV=development
export MICROSCOPE_ENABLED=true
export DATABASE_URL=postgres://user:pass@localhost:5432/mydb?sslmode=disable

make ui-build   # build Vue UI + sync into adaptors
make run        # builds and starts bin/microscope
```

Open `http://127.0.0.1:8093/microscope` (default port for `cmd/server`).

## Option 3 — Embed in your application

Pick the guide for your stack:

- [Go integration](go-integration.md) — `github.com/opentoxic/microscope/adaptor/go`
- [Laravel integration](laravel-integration.md) — `opentoxic/microscope-adaptor-laravel`
- [Python integration](python-integration.md) — `adaptor/python` + `adaptor/django`

For services that cannot embed an adaptor, use a [remote HTTP client](../../clients/).

## Activation

microscope is **active** when all of the following are true:

1. `MICROSCOPE_ENABLED` is `true` (default when unset in Go/PHP)
2. `APP_ENV` is listed in `MICROSCOPE_ALLOWED_ENVS` (default: `development,local`)

See [Configuration](configuration.md) for every variable.

## Verify it works

1. Set env vars and start your app.
2. Make a few HTTP requests to normal routes (not only `/microscope`).
3. Open the dashboard at your configured path (default `/microscope`).
4. Confirm entries appear in the stream and request detail shows payload/headers.

## Next steps

- [Architecture](architecture.md) — mental model for adaptors and shared `core/`
- [Custom events](tutorials/custom-events.md) — mark business moments in the timeline
- [UI development](tutorials/ui-development.md) — change the dashboard and sync assets
