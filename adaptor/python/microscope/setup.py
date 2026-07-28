from __future__ import annotations

import logging
import os
import re
from dataclasses import dataclass
from pathlib import Path

import psycopg

from microscope.config import Config
from microscope.hub import Hub
from microscope.http import ApiRouter, SpaRouter
from microscope.store import MigrationRunner, PostgresStore

_MIGRATIONS_PATH = Path(__file__).resolve().parent / "migrations"
_UI_DIST_PATH = Path(__file__).resolve().parent / "static" / "dist"
_CONNECT_TIMEOUT_SECONDS = 5
_STATEMENT_TIMEOUT_MS = 5000

_log = logging.getLogger("microscope")


def _normalize_dsn(dsn: str) -> str:
    """Strip SQLAlchemy-style driver suffixes (e.g. postgresql+asyncpg://)

    so the DSN is plain libpq syntax that psycopg understands. Apps commonly
    export DATABASE_URL in their ORM's dialect, not psycopg's.
    """
    return re.sub(r"^(postgres(?:ql)?)\+[^:]+://", r"\1://", dsn)


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

    conn = psycopg.connect(
        dsn,
        connect_timeout=_CONNECT_TIMEOUT_SECONDS,
        options=f"-c statement_timeout={_STATEMENT_TIMEOUT_MS}",
    )
    if config.auto_migrate:
        MigrationRunner(conn, _MIGRATIONS_PATH).up()

    store = PostgresStore(conn)
    hub = Hub(store, config)
    api = ApiRouter(hub)
    spa = SpaRouter(_UI_DIST_PATH, config.path_prefix())

    return Microscope(hub=hub, api=api, spa=spa, active=True)


def boot_from_env(app_env: str | None = None) -> Microscope:
    """Boot microscope from the environment. Never raises: any failure

    (unreachable database, bad DSN, migration error) degrades to an
    inactive instance instead of taking the host application down —
    microscope is an optional observability aid, not a hard dependency.
    """
    try:
        dsn = os.environ.get("DATABASE_URL", "")
        if not dsn:
            raise ValueError("DATABASE_URL is required")
        return boot(_normalize_dsn(dsn), app_env or os.environ.get("APP_ENV", "production"))
    except Exception:
        _log.warning("microscope: failed to boot, disabling", exc_info=True)
        return Microscope(hub=None, api=None, spa=None, active=False)
