from __future__ import annotations

import asyncio
import secrets
import threading
import time
import traceback
from typing import Any

from microscope.request_meta import bridge_from_headers, set_request_context


class MicroscopeASGIMiddleware:
    """Pure-ASGI middleware for FastAPI/Starlette apps."""

    def __init__(self, app: Any, microscope: Any) -> None:
        self._app = app
        self._microscope = microscope

    async def __call__(self, scope: dict, receive: Any, send: Any) -> None:
        if scope["type"] != "http" or not self._microscope.active:
            await self._app(scope, receive, send)
            return

        path = scope["path"]
        prefix = self._microscope.hub.config.path_prefix()
        if path == prefix or path.startswith(f"{prefix}/"):
            if path.endswith("/api/stream"):
                await self._handle_stream(send)
                return
            await self._handle_microscope_request(scope, receive, send, path)
            return

        hub = self._microscope.hub
        if hub is None:
            await self._app(scope, receive, send)
            return

        batch_id = secrets.token_hex(16)
        headers = {k.decode("latin-1"): v.decode("latin-1") for k, v in scope.get("headers", [])}
        meta = bridge_from_headers(headers)
        if not meta.request_id:
            meta.request_id = headers.get("x-request-id", "")
        if not meta.correlation_id:
            meta.correlation_id = meta.request_id
        set_request_context(batch_id, meta)

        body = b""
        more_body = True
        while more_body:
            message = await receive()
            body += message.get("body", b"")
            more_body = message.get("more_body", False)
        max_bytes = hub.config.max_body_bytes
        if len(body) > max_bytes:
            body = body[:max_bytes]

        started = time.monotonic()
        status_holder: dict[str, int] = {}
        response_body = bytearray()

        async def send_wrapper(message: dict) -> None:
            if message["type"] == "http.response.start":
                status_holder["status"] = message["status"]
            if message["type"] == "http.response.body":
                chunk = message.get("body", b"")
                if len(response_body) < 65536:
                    response_body.extend(chunk[:65536 - len(response_body)])
            await send(message)

        try:
            async def replay_receive():
                return {"type": "http.request", "body": body, "more_body": False}

            await self._app(scope, replay_receive, send_wrapper)
        except Exception as exc:
            hub.record(
                "exception",
                {
                    "message": str(exc),
                    "path": path,
                    "method": scope["method"],
                    "stack": traceback.format_exc(),
                },
                batch_id=batch_id,
                request_id=meta.request_id,
                correlation_id=meta.correlation_id,
                tags=["panic"],
            )
            raise

        status = status_holder.get("status", 200)
        hub.record(
            "request",
            {
                "method": scope["method"],
                "path": path,
                "query": scope.get("query_string", b"").decode("utf-8", errors="replace"),
                "status": status,
                "duration_ms": round((time.monotonic() - started) * 1000, 2),
                "ip": scope.get("client", ["", 0])[0] if scope.get("client") else "",
                "user_agent": headers.get("user-agent", ""),
                "headers": hub.sanitize_headers({k: [v] for k, v in headers.items()}),
                "request_body": hub.sanitize_json(body),
                "response_body": hub.sanitize_json(bytes(response_body)),
            },
            batch_id=batch_id,
            entry_id=batch_id,
            request_id=meta.request_id,
            correlation_id=meta.correlation_id,
            tags=[f"method:{scope['method']}", f"status:{status}"],
        )

    async def _handle_microscope_request(self, scope: dict, receive: Any, send: Any, path: str) -> None:
        body = b""
        more_body = True
        while more_body:
            message = await receive()
            body += message.get("body", b"")
            more_body = message.get("more_body", False)

        query_string = scope.get("query_string", b"").decode("utf-8")
        result = self._microscope.handle(
            scope["method"], path, body.decode("utf-8", errors="replace"), query_string
        )

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

    async def _handle_stream(self, send: Any) -> None:
        await send(
            {
                "type": "http.response.start",
                "status": 200,
                "headers": [
                    (b"content-type", b"text/event-stream"),
                    (b"cache-control", b"no-cache, no-transform"),
                    (b"connection", b"keep-alive"),
                    (b"x-accel-buffering", b"no"),
                ],
            }
        )

        loop = asyncio.get_running_loop()
        queue: asyncio.Queue[str | None] = asyncio.Queue()

        def write(chunk: str) -> None:
            loop.call_soon_threadsafe(queue.put_nowait, chunk)

        def run_stream() -> None:
            try:
                self._microscope.stream(write)
            finally:
                loop.call_soon_threadsafe(queue.put_nowait, None)

        threading.Thread(target=run_stream, daemon=True).start()

        while True:
            chunk = await queue.get()
            if chunk is None:
                break
            await send({"type": "http.response.body", "body": chunk.encode("utf-8"), "more_body": True})
        await send({"type": "http.response.body", "body": b""})
