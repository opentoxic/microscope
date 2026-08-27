from __future__ import annotations

from contextvars import ContextVar

_batch_id: ContextVar[str] = ContextVar("microscope_batch_id", default="")
_skip_trace: ContextVar[bool] = ContextVar("microscope_skip_trace", default=False)


def with_batch_id(batch_id: str) -> None:
    _batch_id.set(batch_id)


def batch_id_from_context() -> str:
    return _batch_id.get()


def without_trace() -> None:
    _skip_trace.set(True)


def trace_skipped() -> bool:
    return _skip_trace.get()


def is_microscope_path(path: str, prefix: str = "/microscope") -> bool:
    if not prefix:
        prefix = "/microscope"
    return path == prefix or path.startswith(f"{prefix}/")
