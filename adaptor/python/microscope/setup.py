from __future__ import annotations

import os
from dataclasses import dataclass
from pathlib import Path

import psycopg

from microscope.config import Config
from microscope.hub import Hub
from microscope.http import ApiRouter, SpaRouter
from microscope.store import MigrationRunner, PostgresStore

_REPO_ROOT = Path(__file__).resolve().parents[3]
_MIGRATIONS_PATH = _REPO_ROOT / "core" / "migrations"
_UI_DIST_PATH = Path(__file__).resolve().parent / "static" / "dist"


@dataclass
class Microscope:
    hub: Hub | None
    api: ApiRouter | None
    spa: SpaRouter | None
    active: bool

    def handle(self, method: str, path: str, body: str = "", query: str = "") -> dict | None:
        if not self.active or self.api is None or self.spa is None:
            return None

        api_result = self.api.handle(method, path, body, query)
        if api_result.get("status") != 404:
            return api_result

        return self.spa.handle(method, path)


def boot(dsn: str, app_env: str, config: Config | None = None) -> Microscope:
    config = config or Config.from_env()
    if not config.is_active(app_env):
        return Microscope(hub=None, api=None, spa=None, active=False)

    conn = psycopg.connect(dsn)
    if config.auto_migrate:
        MigrationRunner(conn, _MIGRATIONS_PATH).up()

    store = PostgresStore(conn)
    hub = Hub(store, config)
    api = ApiRouter(hub)
    spa = SpaRouter(_UI_DIST_PATH, config.path_prefix())

    return Microscope(hub=hub, api=api, spa=spa, active=True)


def boot_from_env(app_env: str | None = None) -> Microscope:
    dsn = os.environ.get("DATABASE_URL", "")
    if not dsn:
        raise ValueError("DATABASE_URL is required")
    return boot(dsn, app_env or os.environ.get("APP_ENV", "production"))
