# UI development

The dashboard is a Vue 3 + Vite app in [`core/ui`](../../ui/).

## Prerequisites

- Node.js 20+
- [pnpm](https://pnpm.io/) 9+

> **Note:** This project uses pnpm. `npm install` may fail on some Node 24 setups; use pnpm instead.

## Dev server (hot reload)

```bash
cd core/ui
pnpm install
pnpm run dev
```

API calls proxy to `http://127.0.0.1:8093` (override with `HTTP_PORT` env). Use `pnpm run demo` for offline mock data.

## Production build

From repo root:

```bash
make ui-build
```

This runs `pnpm install`, `pnpm run build`, and `sh scripts/sync-core-assets.sh`.

### What sync does

| Source | Destination |
|--------|-------------|
| `core/migrations/` | `adaptor/go/migrations/` |
| `core/ui/dist/` | `adaptor/go/ui/dist/` |
| `core/ui/dist/` | `adaptor/php/resources/ui/dist/` |
| `core/ui/dist/` | `adaptor/python/microscope/static/dist/` |

Go embeds UI and migrations at compile time — **rebuild Go binaries** after `make ui-build`.

PHP/Python serve static files from synced `dist/` directories.

## Project structure

```
core/ui/
├── src/
│   ├── views/          # Entry list, detail, settings
│   ├── components/     # SystemVisuals, AppShell, …
│   ├── api/client.ts   # REST + SSE client
│   └── utils.ts        # Signal labels, metric language detection
├── index.html
├── vite.config.ts      # base: /microscope/
└── package.json
```

## Changing the API

1. Update [`core/api/openapi.yaml`](../../api/openapi.yaml).
2. Implement handlers in each adaptor (Go reference first).
3. Extend `core/ui/src/api/client.ts` if new endpoints are needed.
4. Run Go contract tests: `cd adaptor/go && go test ./core/api/tests/...`

## See also

- [Local development](local-development.md)
- [Architecture](../architecture.md)
