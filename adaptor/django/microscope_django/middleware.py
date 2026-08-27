from __future__ import annotations

import secrets
import time
import traceback
from functools import lru_cache

from django.conf import settings

from microscope.context import is_microscope_path
from microscope.request_meta import django_request_meta, set_request_context
import microscope.setup as microscope_setup

_RESPONSE_BODY_LIMIT = 65536


@lru_cache(maxsize=1)
def get_microscope() -> microscope_setup.Microscope:
    return microscope_setup.boot_from_env()


def _response_body(response) -> bytes:
    if getattr(response, "streaming", False):
        return b""
    if hasattr(response, "render") and callable(response.render):
        if not getattr(response, "is_rendered", True):
            response.render()
    if not hasattr(response, "content"):
        return b""
    content = response.content
    if not isinstance(content, bytes):
        return b""
    return content[:_RESPONSE_BODY_LIMIT]


def _microscope_prefix() -> str:
    prefix = getattr(settings, "MICROSCOPE_PATH", "/microscope")
    if not prefix.startswith("/"):
        prefix = f"/{prefix}"
    return prefix


def _should_record(request) -> bool:
    return not is_microscope_path(request.path, _microscope_prefix())


def _record_request(request, response, hub, batch_id: str, started: float) -> None:
    meta = django_request_meta(request)

    headers: dict[str, list[str]] = {}
    for key, value in request.headers.items():
        headers[key] = [value]

    req_body = getattr(request, "_microscope_request_body", b"")
    resp_body = _response_body(response)

    status = response.status_code
    hub.record(
        "request",
        {
            "method": request.method,
            "path": request.path,
            "query": request.META.get("QUERY_STRING", ""),
            "status": status,
            "duration_ms": round((time.monotonic() - started) * 1000, 2),
            "ip": meta.ip_address,
            "user_agent": meta.user_agent,
            "headers": hub.sanitize_headers(headers),
            "request_body": hub.sanitize_json(req_body),
            "response_body": hub.sanitize_json(resp_body),
        },
        batch_id=batch_id,
        entry_id=batch_id,
        request_id=meta.request_id,
        correlation_id=meta.correlation_id,
        tags=[f"method:{request.method}", f"status:{status}"],
    )


def _record_exception(request, hub, batch_id: str, exception: BaseException) -> None:
    meta = django_request_meta(request)
    hub.record(
        "exception",
        {
            "message": str(exception),
            "path": request.path,
            "method": request.method,
            "stack": traceback.format_exc(),
        },
        batch_id=batch_id,
        request_id=meta.request_id,
        correlation_id=meta.correlation_id,
        tags=["panic"],
    )


class RecordRequestsMiddleware:
    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        microscope = get_microscope()
        hub = microscope.hub
        if hub is None or not _should_record(request):
            return self.get_response(request)

        batch_id = secrets.token_hex(16)
        meta = django_request_meta(request)
        set_request_context(batch_id, meta)
        request._microscope_batch_id = batch_id
        request._microscope_started = time.monotonic()

        max_bytes = hub.config.max_body_bytes
        request._microscope_request_body = request.body[:max_bytes] if hasattr(request, "body") else b""

        started = request._microscope_started
        try:
            response = self.get_response(request)
        except Exception as exc:
            _record_exception(request, hub, batch_id, exc)
            raise

        if not _should_record(request):
            return response

        _record_request(request, response, hub, batch_id, started)
        return response
