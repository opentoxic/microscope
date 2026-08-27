from __future__ import annotations

import secrets
import time
import traceback
from functools import lru_cache

from django.conf import settings
from django.utils.deprecation import MiddlewareMixin

from microscope.context import is_microscope_path
from microscope.request_meta import django_request_meta, set_request_context
import microscope.setup as microscope_setup


@lru_cache(maxsize=1)
def get_microscope() -> microscope_setup.Microscope:
    return microscope_setup.boot_from_env()


class RecordRequestsMiddleware(MiddlewareMixin):
    def process_request(self, request):
        microscope = get_microscope()
        if microscope.hub is None:
            return None

        prefix = getattr(settings, "MICROSCOPE_PATH", "/microscope")
        if not prefix.startswith("/"):
            prefix = f"/{prefix}"
        if is_microscope_path(request.path, prefix):
            return None

        batch_id = secrets.token_hex(16)
        meta = django_request_meta(request)
        set_request_context(batch_id, meta)
        request._microscope_batch_id = batch_id
        request._microscope_started = time.monotonic()
        return None

    def process_response(self, request, response):
        microscope = get_microscope()
        hub = microscope.hub
        if hub is None or not hasattr(request, "_microscope_batch_id"):
            return response

        prefix = getattr(settings, "MICROSCOPE_PATH", "/microscope")
        if not prefix.startswith("/"):
            prefix = f"/{prefix}"
        if is_microscope_path(request.path, prefix):
            return response

        started = getattr(request, "_microscope_started", time.monotonic())
        batch_id = request._microscope_batch_id
        meta = django_request_meta(request)

        headers: dict[str, list[str]] = {}
        for key, value in request.headers.items():
            headers[key] = [value]

        max_bytes = hub.config.max_body_bytes
        req_body = request.body[:max_bytes] if hasattr(request, "body") else b""
        resp_body = b""
        if hasattr(response, "content"):
            resp_body = bytes(response.content[:65536])

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
        return response

    def process_exception(self, request, exception):
        microscope = get_microscope()
        hub = microscope.hub
        if hub is None or not hasattr(request, "_microscope_batch_id"):
            return None

        prefix = getattr(settings, "MICROSCOPE_PATH", "/microscope")
        if not prefix.startswith("/"):
            prefix = f"/{prefix}"
        if is_microscope_path(request.path, prefix):
            return None

        meta = django_request_meta(request)
        hub.record(
            "exception",
            {
                "message": str(exception),
                "path": request.path,
                "method": request.method,
                "stack": traceback.format_exc(),
            },
            batch_id=request._microscope_batch_id,
            request_id=meta.request_id,
            correlation_id=meta.correlation_id,
            tags=["panic"],
        )
        return None
