# Python / Django integration

Native Python adaptor with optional Django glue. Runs in-process against PostgreSQL.

## Install

From this monorepo:

```bash
pip install -e adaptor/python
pip install -e adaptor/django
```

## Django setup

`settings.py`:

```python
INSTALLED_APPS += ["microscope_django"]

MIDDLEWARE += [
    "microscope_django.middleware.RecordRequestsMiddleware",
]

# Same database as your app
MICROSCOPE_ENABLED = True
MICROSCOPE_PATH = "/microscope"
```

`urls.py`:

```python
from django.urls import include, path

urlpatterns += [
    path("microscope/", include("microscope_django.urls")),
]
```

Environment variables (see [Configuration](configuration.md)) are also supported:

```bash
export APP_ENV=development
export MICROSCOPE_ENABLED=true
export DATABASE_URL=postgres://...
```

## ASGI / FastAPI (without Django)

Use the Python package directly — see `adaptor/python/microscope/` for `Setup`, hub, and HTTP handlers. Django is optional convenience.

## Migrations

When `MICROSCOPE_AUTO_MIGRATE=true`, migrations run on boot. Schema matches `core/migrations/`.

## Runtime metrics

Python runtime metrics follow the same `python.runtime` naming convention as other SDKs. Wire periodic sampling in your app bootstrap or use a remote [TypeScript client](../../clients/typescript/) for Node services.

## Verify

1. Start Django (`manage.py runserver`).
2. Hit application routes.
3. Open `http://127.0.0.1:8000/microscope/`.

## See also

- [Getting started](getting-started.md)
- [Architecture](architecture.md)
- [Custom events](tutorials/custom-events.md)
