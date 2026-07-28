from __future__ import annotations

import json
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import psycopg

from microscope.entry import ALL_ENTRY_TYPES, Entry


class PostgresStore:
    def __init__(self, conn: psycopg.Connection) -> None:
        self.conn = conn

    def insert(self, entry: Entry) -> None:
        with self.conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO microscope_entries
                (id, batch_id, type, request_id, correlation_id, tags, content, created_at)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
                """,
                (
                    entry.id,
                    entry.batch_id,
                    entry.type,
                    entry.request_id or None,
                    entry.correlation_id or None,
                    json.dumps(entry.tags),
                    json.dumps(entry.content),
                    entry.created_at,
                ),
            )
        self.conn.commit()

    def get(self, entry_id: str) -> Entry | None:
        with self.conn.cursor() as cur:
            cur.execute(
                "SELECT id, batch_id, type, request_id, correlation_id, tags, content, created_at "
                "FROM microscope_entries WHERE id = %s",
                (entry_id,),
            )
            row = cur.fetchone()
        return self._map_row(row) if row else None

    def list_entries(
        self,
        entry_type: str | None,
        search: str | None,
        limit: int,
        offset: int,
    ) -> dict[str, Any]:
        limit = max(1, min(limit or 50, 200))
        where = "WHERE 1=1"
        params: list[Any] = []
        if entry_type:
            where += " AND type = %s"
            params.append(entry_type)
        if search:
            where += " AND (content::text ILIKE %s OR request_id ILIKE %s)"
            params.extend([f"%{search}%", f"%{search}%"])
        with self.conn.cursor() as cur:
            cur.execute(f"SELECT COUNT(*) FROM microscope_entries {where}", params)
            total = int(cur.fetchone()[0])
            cur.execute(
                f"SELECT id, batch_id, type, request_id, correlation_id, tags, content, created_at "
                f"FROM microscope_entries {where} ORDER BY created_at DESC LIMIT %s OFFSET %s",
                [*params, limit, max(0, offset)],
            )
            rows = cur.fetchall()
        return {"entries": [self._map_row(r).to_dict() for r in rows], "total": total}

    def list_by_batch(self, batch_id: str) -> list[Entry]:
        with self.conn.cursor() as cur:
            cur.execute(
                "SELECT id, batch_id, type, request_id, correlation_id, tags, content, created_at "
                "FROM microscope_entries WHERE batch_id = %s ORDER BY created_at ASC",
                (batch_id,),
            )
            rows = cur.fetchall()
        return [self._map_row(r) for r in rows]

    def clear_all(self) -> int:
        with self.conn.cursor() as cur:
            cur.execute("DELETE FROM microscope_entries")
            deleted = cur.rowcount
            cur.execute("VACUUM FULL ANALYZE microscope_entries")
        self.conn.commit()
        return deleted

    def list_type_settings(self) -> list[dict[str, Any]]:
        with self.conn.cursor() as cur:
            cur.execute(
                """
                SELECT configured.type,
                       COALESCE(settings.enabled, TRUE),
                       COALESCE(entries.count, 0)
                FROM unnest(%s::text[]) AS configured(type)
                LEFT JOIN microscope_settings settings ON settings.type = configured.type
                LEFT JOIN (
                    SELECT type, COUNT(*) AS count FROM microscope_entries GROUP BY type
                ) entries ON entries.type = configured.type
                """,
                (ALL_ENTRY_TYPES,),
            )
            rows = cur.fetchall()
        return [{"type": r[0], "enabled": bool(r[1]), "count": int(r[2])} for r in rows]

    def set_type_enabled(self, entry_type: str, enabled: bool) -> int:
        deleted = 0
        with self.conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO microscope_settings (type, enabled, updated_at)
                VALUES (%s, %s, NOW())
                ON CONFLICT (type) DO UPDATE SET enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at
                """,
                (entry_type, enabled),
            )
            if not enabled:
                cur.execute("DELETE FROM microscope_entries WHERE type = %s", (entry_type,))
                deleted = cur.rowcount
        self.conn.commit()
        return deleted

    def get_option(self, key: str) -> str | None:
        with self.conn.cursor() as cur:
            cur.execute("SELECT value FROM microscope_options WHERE key = %s", (key,))
            row = cur.fetchone()
        return None if row is None else json.dumps(row[0])

    def set_option(self, key: str, value: str) -> None:
        with self.conn.cursor() as cur:
            cur.execute(
                """
                INSERT INTO microscope_options (key, value, updated_at)
                VALUES (%s, %s::jsonb, NOW())
                ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
                """,
                (key, value),
            )
        self.conn.commit()

    def storage_usage(self) -> dict[str, float | int]:
        with self.conn.cursor() as cur:
            cur.execute(
                """
                SELECT
                    ROUND(pg_total_relation_size('microscope_entries') / 1048576.0, 2),
                    ROUND(pg_relation_size('microscope_entries') / 1048576.0, 2),
                    ROUND((pg_total_relation_size('microscope_entries') - pg_relation_size('microscope_entries')) / 1048576.0, 2),
                    ROUND(pg_total_relation_size('microscope_settings') / 1048576.0, 2),
                    ROUND(COALESCE((SELECT pg_total_relation_size(oid) FROM pg_class WHERE relname = 'microscope_schema_migrations'), 0) / 1048576.0, 2),
                    (SELECT COUNT(*) FROM microscope_entries)
                """
            )
            row = cur.fetchone()
        entries_mb = float(row[0])
        settings_mb = float(row[3])
        migrations_mb = float(row[4])
        return {
            "entries_mb": entries_mb,
            "entries_data_mb": float(row[1]),
            "entries_indexes_mb": float(row[2]),
            "settings_mb": settings_mb,
            "migrations_mb": migrations_mb,
            "total_mb": entries_mb + settings_mb + migrations_mb,
            "entry_count": int(row[5]),
        }

    def _map_row(self, row: tuple[Any, ...]) -> Entry:
        tags = row[5] if isinstance(row[5], list) else json.loads(row[5])
        content = row[6] if isinstance(row[6], dict) else json.loads(row[6])
        created = row[7]
        if isinstance(created, datetime):
            created_at = created.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
        else:
            created_at = str(created)
        return Entry(
            id=row[0],
            batch_id=row[1],
            type=row[2],
            content=content,
            created_at=created_at,
            request_id=row[3] or "",
            correlation_id=row[4] or "",
            tags=tags,
        )


class MigrationRunner:
    FILES = [
        "001_microscope.up.sql",
        "002_microscope_settings.up.sql",
        "003_microscope_options.up.sql",
    ]

    def __init__(self, conn: psycopg.Connection, migrations_path: Path) -> None:
        self.conn = conn
        self.migrations_path = migrations_path

    def up(self) -> None:
        with self.conn.cursor() as cur:
            cur.execute(
                """
                CREATE TABLE IF NOT EXISTS microscope_schema_migrations (
                    version TEXT PRIMARY KEY,
                    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
                )
                """
            )
            for file in self.FILES:
                version = file.replace(".up.sql", "")
                cur.execute(
                    "SELECT 1 FROM microscope_schema_migrations WHERE version = %s",
                    (version,),
                )
                if cur.fetchone():
                    continue
                sql = (self.migrations_path / file).read_text(encoding="utf-8")
                cur.execute(sql)
                cur.execute(
                    "INSERT INTO microscope_schema_migrations (version) VALUES (%s)",
                    (version,),
                )
        self.conn.commit()
