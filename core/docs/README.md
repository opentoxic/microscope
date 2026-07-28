# microscope documentation

Development observability that runs **in-process** with your app — no sidecar required.

## Start here

| Guide | When to read |
|-------|----------------|
| [Getting started](getting-started.md) | First-time setup, demo mode, standalone server |
| [Configuration](configuration.md) | Environment variables and activation rules |
| [Architecture](architecture.md) | How `core/`, adaptors, and clients fit together |

## Integration guides

| Stack | Guide |
|-------|--------|
| Go (`net/http`, chi, gin, …) | [Go integration](go-integration.md) |
| Laravel | [Laravel integration](laravel-integration.md) |
| Python / Django | [Python integration](python-integration.md) |
| Remote services (Node, Ruby, Elixir) | [clients/](../../clients/) READMEs |

## Tutorials

| Topic | Guide |
|-------|--------|
| Build and sync the dashboard UI | [UI development](tutorials/ui-development.md) |
| Record custom business events | [Custom events](tutorials/custom-events.md) |
| Work on this repository | [Local development](tutorials/local-development.md) |

## Reference

- [OpenAPI contract](../api/openapi.yaml) — shared HTTP API
- [Database migrations](../migrations/) — PostgreSQL schema
- [Dashboard source](../ui/) — Vue 3 SPA
