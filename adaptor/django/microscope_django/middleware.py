from __future__ import annotations

import time
from functools import lru_cache

from django.conf import settings
from django.utils.deprecation import MiddlewareMixin

from microscope.runtime_metrics import sample
from microscope.setup import Microscope, boot_from_env


@lru_cache(maxsize=1)
def get_microscope() -> Microscope:
    return boot_from_env()


class RecordRequestsMiddleware(MiddlewareMixin):
    def process_request(self, request):
        request._microscope_started = time.monotonic()

    def process_response(self, request, response):
        microscope = get_microscope()
        hub = microscope.hub
        if hub is None:
            return response

        path = request.path.lstrip("/")
        microscope_path = getattr(settings, "MICROSCOPE_PATH", "microscope").strip("/")
        if path == microscope_path or path.startswith(f"{microscope_path}/"):
            return response

        started = getattr(request, "_microscope_started", time.monotonic())
        hub.record(
            "request",
            {
                "method": request.method,
                "path": request.path,
                "status": response.status_code,
                "duration_ms": round((time.monotonic() - started) * 1000, 2),
            },
        )
        hub.record("metric", sample())
        return response
