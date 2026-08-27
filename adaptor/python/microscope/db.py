from __future__ import annotations

import time
from datetime import datetime, timezone
from typing import Any

from microscope.context import batch_id_from_context, trace_skipped
from microscope.entry import Entry
from microscope.hub import Hub
from microscope.request_meta import request_meta_from_context


class QueryTracer:
    def __init__(self) -> None:
        self._hub: Hub | None = None

    def bind(self, hub: Hub) -> None:
        self._hub = hub

    def execute_wrapper(self, execute, sql, params, many, context):
        if trace_skipped() or self._hub is None or "microscope_entries" in sql.lower():
            return execute(sql, params, many, context)
        started = time.monotonic()
        error: Exception | None = None
        try:
            return execute(sql, params, many, context)
        except Exception as exc:
            error = exc
            raise
        finally:
            duration_ms = (time.monotonic() - started) * 1000
            content: dict[str, Any] = {
                "sql": sql,
                "args": self._hub.sanitize_args(list(params or [])),
                "duration_ms": duration_ms,
            }
            if error is not None:
                content["error"] = str(error)
            meta = request_meta_from_context()
            entry = Entry(
                id="",
                batch_id=batch_id_from_context(),
                type="query",
                content=content,
                created_at=datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
                request_id=meta.request_id,
                correlation_id=meta.correlation_id,
                tags=["sql"],
            )
            self._hub.record_entry(entry)
