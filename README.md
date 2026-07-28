<div align="center">

<img src="./banner.svg" alt="microscope" width="100%" />

<br/>

![go](https://img.shields.io/badge/go-in--process-20d9ee?style=for-the-badge&labelColor=0a1218)
![php](https://img.shields.io/badge/laravel·php-native-9b9de2?style=for-the-badge&labelColor=0a1218)
![python](https://img.shields.io/badge/python·django-ffd343?style=for-the-badge&labelColor=0a1218)
![vue](https://img.shields.io/badge/ui-vue%203-28e0a0?style=for-the-badge&labelColor=0a1218)
![postgres](https://img.shields.io/badge/storage-postgresql-4c9fff?style=for-the-badge&labelColor=0a1218)

**Development observability for any stack — in-process, no sidecar.**

[Getting started](core/docs/getting-started.md) · [Docs](core/docs/README.md) · [API](core/api/openapi.yaml)

</div>

<br/>

## What is microscope?

microscope records what your application does during development: HTTP requests and responses,
SQL queries, structured logs, Redis, queues, schedules, outgoing HTTP, runtime metrics, custom
events, and panics. A Vue 3 dashboard is served from the same process at `/microscope`.

Each **language adaptor** embeds recording and the UI natively — your app and microscope share
one PostgreSQL database. No separate Docker collector required.

<p align="center">
  <img src="./core/docs/assets/overview.png" alt="Microscope runtime overview dashboard" width="96%" />
</p>

<p align="center"><em>Live signal stream, runtime vitals, and service topology</em></p>

## Features

- **In-process** — Go, Laravel/PHP, Python/Django adaptors; remote clients for Node, Ruby, Elixir
- **Full HTTP capture** — method, path, status, headers, query, request/response bodies
- **SQL tracing** — queries linked to the request that triggered them
- **Runtime topology** — Postgres, Redis, Kafka, outbound HTTP visualized from live signals
- **Live stream** — SSE feed; pause recording; per-signal-type toggles
- **Dev-safe defaults** — active only in `development` / `local` unless you configure otherwise

## Quick start

**UI demo** (no database):

```bash
cd core/ui && pnpm install && pnpm run demo
# → http://127.0.0.1:5173/microscope/
```

**Standalone server** (this repo):

```bash
export APP_ENV=development MICROSCOPE_ENABLED=true
export DATABASE_URL=postgres://user:pass@localhost:5432/mydb?sslmode=disable

make ui-build && make run
# → http://127.0.0.1:8093/microscope
```

**Embed in your app** — pick a guide:

| Stack | Guide |
|-------|--------|
| Go | [go-integration.md](core/docs/go-integration.md) |
| Laravel | [laravel-integration.md](core/docs/laravel-integration.md) |
| Python / Django | [python-integration.md](core/docs/python-integration.md) |
| Node / Ruby / Elixir (remote) | [clients/](clients/) |

Full walkthrough: **[Getting started](core/docs/getting-started.md)**

## Repository layout

| Path | Purpose |
|------|---------|
| [`core/ui`](core/ui) | Vue 3 dashboard (`make ui-build`) |
| [`core/migrations`](core/migrations) | Canonical PostgreSQL schema |
| [`core/api`](core/api) | OpenAPI contract |
| [`core/docs`](core/docs) | Documentation and tutorials |
| [`adaptor/go`](adaptor/go) | Go module (reference implementation) |
| [`adaptor/php`](adaptor/php) | Native PHP core |
| [`adaptor/laravel`](adaptor/laravel) | Laravel provider and middleware |
| [`adaptor/python`](adaptor/python) | Python core |
| [`adaptor/django`](adaptor/django) | Django URLs and middleware |
| [`clients/`](clients/) | HTTP clients for remote recording |
| [`scripts/sync-core-assets.sh`](scripts/sync-core-assets.sh) | Sync migrations + UI into adaptors |

See [Architecture](core/docs/architecture.md) for how the pieces connect.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `MICROSCOPE_ENABLED` | `true` | Master switch |
| `MICROSCOPE_PATH` | `/microscope` | Dashboard + API prefix |
| `MICROSCOPE_ALLOWED_ENVS` | `development,local` | When microscope may run |
| `MICROSCOPE_RETENTION_HOURS` | `24` | Auto-prune window |
| `MICROSCOPE_MAX_BODY_BYTES` | `65536` | Max captured body size |
| `MICROSCOPE_AUTO_MIGRATE` | `true` | Run migrations on boot |
| `MICROSCOPE_REDACT_SENSITIVE` | `false` | Mask secrets before storage |

Active when `MICROSCOPE_ENABLED` is true **and** `APP_ENV` is allowed. Details: [Configuration](core/docs/configuration.md).

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/microscope/api/entries` | List entries |
| GET | `/microscope/api/entries/{id}` | Entry detail |
| GET | `/microscope/api/stream` | Live SSE stream |
| POST | `/microscope/api/entries` | [Custom event](core/docs/tutorials/custom-events.md) |
| GET/PUT | `/microscope/api/recording` | Pause / resume |
| GET/PUT | `/microscope/api/redaction` | Redaction policy |
| GET | `/microscope/api/settings` | Signal type policies |

Full schema: [`core/api/openapi.yaml`](core/api/openapi.yaml)

## Tutorials

- [UI development](core/docs/tutorials/ui-development.md) — build and sync the dashboard
- [Custom events](core/docs/tutorials/custom-events.md) — mark business moments
- [Local development](core/docs/tutorials/local-development.md) — contribute to this repo

## Development

```bash
make test-all          # go + php + python
make ui-build          # pnpm build + sync-core
cd core/ui && pnpm run dev
```

Requires Go 1.25+, PHP 8.1+, Python 3.10+, Node 20+, pnpm.

## License

MIT
