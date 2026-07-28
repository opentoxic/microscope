# Architecture

microscope is a **monorepo** with shared assets and per-language adaptors. Each adaptor runs
observability **in-process**: your app and the dashboard share one PostgreSQL database.

```
┌─────────────────────────────────────────────────────────────┐
│  Your application process                                   │
│  ┌──────────────┐   records    ┌─────────────────────────┐  │
│  │ HTTP / SQL / │ ──────────► │ Hub + Store (adaptor)   │  │
│  │ Redis / jobs │             │                         │  │
│  └──────────────┘             │  ┌───────────────────┐  │  │
│                               │  │ Vue dashboard SPA │  │  │
│  ┌──────────────┐   serves    │  │ + REST/SSE API    │  │  │
│  │   Browser    │ ◄────────── │  └───────────────────┘  │  │
│  └──────────────┘             └───────────┬─────────────┘  │
└───────────────────────────────────────────┼────────────────┘
                                            │
                                            ▼
                                   ┌─────────────────┐
                                   │   PostgreSQL    │
                                   │ microscope_*    │
                                   └─────────────────┘
```

## Repository layout

| Path | Role |
|------|------|
| [`core/migrations`](../migrations/) | Canonical SQL schema (synced into Go embeds) |
| [`core/api/openapi.yaml`](../api/openapi.yaml) | Shared HTTP API contract |
| [`core/ui`](../ui/) | Vue 3 dashboard source |
| [`core/docs`](.) | Documentation |
| [`adaptor/go`](../../adaptor/go/) | Reference implementation (Go module) |
| [`adaptor/php`](../../adaptor/php/) | Native PHP hub, store, API, SPA serving |
| [`adaptor/laravel`](../../adaptor/laravel/) | Laravel provider, middleware, migrations |
| [`adaptor/python`](../../adaptor/python/) | Python hub and ASGI handlers |
| [`adaptor/django`](../../adaptor/django/) | Django URLs and middleware |
| [`clients/`](../../clients/) | Thin HTTP clients for remote recording |
| [`scripts/sync-core-assets.sh`](../../scripts/sync-core-assets.sh) | Copy migrations + built UI into adaptors |

There is **one** UI source tree: `core/ui`. Built assets are copied into each adaptor before release builds.

## Signal types

| Type | Source |
|------|--------|
| `request` | Incoming HTTP (middleware) |
| `query` | SQL (`pgx` tracer, Laravel `DB::listen`, …) |
| `log` | Structured logs (Go `slog` tee) |
| `redis` | Redis commands (go-redis hook) |
| `job` / `schedule` | Queue and cron instrumentation |
| `http-client` | Outgoing HTTP |
| `metric` | Runtime samples (`go.runtime`, `php.runtime`, …) |
| `custom` | Your code via API or SDK |
| `exception` | Panics and unhandled errors |

## Go adaptor (reference)

The Go adaptor is the reference for API behaviour and contract tests. Other adaptors implement the same OpenAPI surface and serve the same embedded UI.

Typical integration:

1. `microscope.Setup()` — pool, hub, optional Redis/Kafka hooks
2. `ms.RegisterRoutes(mux)` — mount API + SPA on your router
3. `ms.HTTPHandler(app, opts)` — wrap app with capture middleware

See [Go integration](go-integration.md).

## PHP / Laravel stack

```
adaptor/laravel  →  adaptor/php  →  PostgreSQL
     │                    │
     │                    └── Hub, PostgresStore, ApiRouter, SpaRouter
     └── ServiceProvider, RecordRequests middleware, DB::listen
```

Laravel does **not** proxy to an external Go service. The PHP core runs natively.

## Python / Django stack

```
adaptor/django  →  adaptor/python  →  PostgreSQL
```

Django provides URL mounting and request middleware; Python core handles storage and API.

## Remote clients

When a service cannot embed an adaptor (browser-only app, legacy binary, separate team repo),
use [`clients/`](../../clients/) to POST custom events and runtime metrics to an existing
microscope HTTP endpoint.

## Contract testing

Go contract tests in `adaptor/go/core/api/tests/` verify handler routes match `core/api/openapi.yaml`. Keep adaptors aligned when extending the API.
