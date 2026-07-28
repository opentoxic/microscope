from __future__ import annotations

import time
from typing import Any

from microscope.runtime_metrics import sample
from microscope.setup import Microscope


class MicroscopeASGIMiddleware:
    """Pure-ASGI middleware for FastAPI/Starlette apps.

    Records each HTTP request/response and mounts the microscope API + dashboard
    under ``microscope.config.path_prefix()``. No Starlette import required, so
    this works with any ASGI framework, not just FastAPI.
    """

    def __init__(self, app: Any, microscope: Microscope) -> None:
        self._app = app
        self._microscope = microscope

    async def __call__(self, scope: dict, receive: Any, send: Any) -> None:
        if scope["type"] != "http" or not self._microscope.active:
            await self._app(scope, receive, send)
            return

        path = scope["path"]
        prefix = self._microscope.hub.config.path_prefix()
        if path == prefix or path.startswith(f"{prefix}/"):
            await self._handle_microscope_request(scope, receive, send, path)
            return

        started = time.monotonic()
        status_holder: dict[str, int] = {}

        async def send_wrapper(message: dict) -> None:
            if message["type"] == "http.response.start":
                status_holder["status"] = message["status"]
            await send(message)

        await self._app(scope, receive, send_wrapper)

        hub = self._microscope.hub
        if hub is not None:
            hub.record(
                "request",
                {
                    "method": scope["method"],
                    "path": path,
                    "status": status_holder.get("status", 0),
                    "duration_ms": round((time.monotonic() - started) * 1000, 2),
                },
            )
            hub.record("metric", sample())

    async def _handle_microscope_request(self, scope: dict, receive: Any, send: Any, path: str) -> None:
        body = b""
        more_body = True
        while more_body:
            message = await receive()
            body += message.get("body", b"")
            more_body = message.get("more_body", False)

        query_string = scope.get("query_string", b"").decode("utf-8")
        result = self._microscope.handle(scope["method"], path, body.decode("utf-8", errors="replace"), query_string)

        if result is None:
            await send({"type": "http.response.start", "status": 404, "headers": []})
            await send({"type": "http.response.body", "body": b""})
            return

        headers = [
            (key.lower().encode("utf-8"), value.encode("utf-8"))
            for key, value in result.get("headers", {}).items()
        ]
        response_body = result.get("body", "")
        if isinstance(response_body, str):
            response_body = response_body.encode("utf-8")

        await send({"type": "http.response.start", "status": result["status"], "headers": headers})
        await send({"type": "http.response.body", "body": response_body})
