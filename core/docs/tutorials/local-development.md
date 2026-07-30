# Local development

Guide for contributors working on the microscope monorepo.

## Clone and test

```bash
git clone https://github.com/opentoxic/microscope.git
cd microscope

# Go tests (after sync-core)
make sync-core
make test

# PHP
make test-php

# Python
make test-python

# All
make test-all
```

## Build UI

```bash
make ui-build    # pnpm build + sync assets
make ui-dev      # Vite dev server
```

Requires pnpm. See [UI development](ui-development.md).

## Run standalone server

```bash
export APP_ENV=development
export MICROSCOPE_ENABLED=true
export MICROSCOPE_DATABASE_URL=postgres://user:pass@localhost:5432/microscope?sslmode=disable

# Or copy .env.example to .env — make run loads .env automatically.

make run
```

## Makefile targets

| Target | Description |
|--------|-------------|
| `sync-core` | Copy migrations + UI dist into adaptors |
| `ui-install` | `pnpm install` in `core/ui` |
| `ui-build` | Build UI and sync |
| `ui-dev` | Vite dev server |
| `build` | Build Go `cmd/server` binary |
| `run` | Build and run standalone server |
| `test` | Go unit tests |
| `test-php` | PHPUnit |
| `test-python` | pytest |
| `test-all` | All of the above |

## CI

GitHub Actions (`.github/workflows/ci.yml`) runs Go, PHP, and Python tests on Ubuntu after `sh scripts/sync-core-assets.sh`.

## Adding a signal type

1. Define the type in Go (`adaptor/go/entry.go`) and PHP (`EntryType.php`).
2. Add collector hooks in each adaptor.
3. Update OpenAPI and UI signal registry (`core/ui/src/utils.ts`).
4. Add contract test coverage in Go.

## Path conventions

- **Single UI source:** `core/ui` only (no duplicate `ui/` at repo root).
- **Single migration source:** `core/migrations` → synced to Go embed path.
- **Adaptors** implement the same API; Go is the reference.

## See also

- [Architecture](../architecture.md)
- [Getting started](../getting-started.md)
