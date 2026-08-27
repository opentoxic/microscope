from __future__ import annotations

from contextvars import ContextVar
from dataclasses import dataclass
from typing import Any

from microscope.context import with_batch_id


@dataclass
class RequestMeta:
    request_id: str = ""
    correlation_id: str = ""
    ip_address: str = ""
    user_agent: str = ""


_request_meta: ContextVar[RequestMeta] = ContextVar(
    "microscope_request_meta", default=RequestMeta()
)


def with_request_meta(meta: RequestMeta) -> None:
    _request_meta.set(meta)


def request_meta_from_context() -> RequestMeta:
    return _request_meta.get()


def bridge_from_headers(headers: dict[str, str]) -> RequestMeta:
    lowered = {k.lower(): v for k, v in headers.items()}
    request_id = lowered.get("x-request-id", "")
    correlation_id = (
        lowered.get("x-correlation-id", "")
        or lowered.get("x-qobly-correlation-id", "")
        or request_id
    )
    return RequestMeta(
        request_id=request_id,
        correlation_id=correlation_id,
        ip_address="",
        user_agent=lowered.get("user-agent", ""),
    )


def set_request_context(batch_id: str, meta: RequestMeta | None = None) -> None:
    with_batch_id(batch_id)
    if meta is not None:
        with_request_meta(meta)


def _request_attr(request: Any, name: str) -> str:
    value = getattr(request, name, "")
    return value if isinstance(value, str) else ""


def django_request_meta(request: Any) -> RequestMeta:
    headers = {
        k.replace("HTTP_", "").replace("_", "-"): v
        for k, v in request.META.items()
        if k.startswith("HTTP_")
    }
    meta = bridge_from_headers(headers)
    meta.ip_address = request.META.get("REMOTE_ADDR", "")
    if not meta.user_agent:
        meta.user_agent = request.META.get("HTTP_USER_AGENT", "")

    # Prefer host-assigned IDs (e.g. Qobly RequestContextMiddleware) over headers.
    meta.request_id = (
        _request_attr(request, "request_id")
        or meta.request_id
        or request.META.get("HTTP_X_REQUEST_ID", "")
    )
    meta.correlation_id = (
        _request_attr(request, "correlation_id")
        or meta.correlation_id
        or request.META.get("HTTP_X_CORRELATION_ID", "")
        or request.META.get("HTTP_X_QOBLY_CORRELATION_ID", "")
        or meta.request_id
    )
    return meta
