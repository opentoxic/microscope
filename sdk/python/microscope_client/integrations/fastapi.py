"""FastAPI / Starlette middleware that records each request as a microscope entry.

    from fastapi import FastAPI
    from microscope_client.integrations.fastapi import MicroscopeMiddleware

    app = FastAPI()
    app.add_middleware(MicroscopeMiddleware, base_url="http://localhost:8093/microscope")
"""

from __future__ import annotations

import time

from starlette.middleware.base import BaseHTTPMiddleware, RequestResponseEndpoint
from starlette.requests import Request
from starlette.responses import Response

from microscope_client import MicroscopeClient


class MicroscopeMiddleware(BaseHTTPMiddleware):
    def __init__(self, app, base_url: str) -> None:
        super().__init__(app)
        self.client = MicroscopeClient(base_url=base_url)

    async def dispatch(self, request: Request, call_next: RequestResponseEndpoint) -> Response:
        started = time.monotonic()
        response = await call_next(request)
        self.client.record(
            "http_request",
            content={
                "method": request.method,
                "path": request.url.path,
                "status": response.status_code,
                "duration_ms": round((time.monotonic() - started) * 1000, 2),
            },
        )
        return response
