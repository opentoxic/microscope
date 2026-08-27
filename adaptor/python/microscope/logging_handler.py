from __future__ import annotations

import logging
from typing import Any

from microscope.context import batch_id_from_context, is_microscope_path
from microscope.entry import Entry
from microscope.hub import Hub
from microscope.request_meta import request_meta_from_context


class MicroscopeLogHandler(logging.Handler):
    """Tee log records to microscope as log signal entries."""

    def __init__(self, hub: Hub, inner: logging.Handler | None = None) -> None:
        super().__init__()
        self._hub = hub
        self._inner = inner
        self._prefix = hub.config.path_prefix()

    def emit(self, record: logging.LogRecord) -> None:
        if self._skip_stdout(record):
            return
        if not self._skip_recording(record):
            meta = request_meta_from_context()
            attrs: dict[str, Any] = {
                "level": record.levelname,
                "message": record.getMessage(),
            }
            for key, value in record.__dict__.items():
                if key.startswith("_") or key in {
                    "name",
                    "msg",
                    "args",
                    "created",
                    "filename",
                    "funcName",
                    "levelno",
                    "lineno",
                    "module",
                    "msecs",
                    "pathname",
                    "process",
                    "processName",
                    "relativeCreated",
                    "stack_info",
                    "exc_info",
                    "exc_text",
                    "thread",
                    "threadName",
                    "taskName",
                    "message",
                }:
                    continue
                attrs[key] = value
            entry = Entry(
                id="",
                batch_id=batch_id_from_context(),
                type="log",
                content=self._hub.sanitize_map(attrs) or {},
                created_at=self._utc_now(record),
                request_id=meta.request_id,
                correlation_id=meta.correlation_id,
                tags=[f"level:{record.levelname}"],
            )
            self._hub.record_entry(entry)
        if self._inner is not None:
            self._inner.emit(record)

    def _skip_recording(self, record: logging.LogRecord) -> bool:
        return record.getMessage() == "request" or self._skip_microscope(record)

    def _skip_stdout(self, record: logging.LogRecord) -> bool:
        return self._skip_microscope(record)

    def _skip_microscope(self, record: logging.LogRecord) -> bool:
        message = record.getMessage()
        if message.startswith("microscope:"):
            return True
        if message != "request":
            return False
        path = getattr(record, "path", "")
        return is_microscope_path(str(path), self._prefix)

    @staticmethod
    def _utc_now(record: logging.LogRecord) -> str:
        from datetime import datetime, timezone

        return datetime.fromtimestamp(record.created, tz=timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
