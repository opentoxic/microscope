"""Django middleware that records each request as a microscope entry.

Add to settings.py:

    MICROSCOPE_BASE_URL = "http://localhost:8093/microscope"
    MIDDLEWARE = [
        "microscope_client.integrations.django.MicroscopeMiddleware",
        ...
    ]
"""

from __future__ import annotations

import time

from django.conf import settings

from microscope_client import MicroscopeClient

_client: MicroscopeClient | None = None


def _get_client() -> MicroscopeClient:
    global _client
    if _client is None:
        _client = MicroscopeClient(base_url=settings.MICROSCOPE_BASE_URL)
    return _client


class MicroscopeMiddleware:
    def __init__(self, get_response):
        self.get_response = get_response
        self.client = _get_client()

    def __call__(self, request):
        started = time.monotonic()
        response = self.get_response(request)
        self.client.record(
            "http_request",
            content={
                "method": request.method,
                "path": request.path,
                "status": response.status_code,
                "duration_ms": round((time.monotonic() - started) * 1000, 2),
            },
        )
        return response
