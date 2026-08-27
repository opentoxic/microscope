from __future__ import annotations

import queue
import threading

from django.http import HttpRequest, HttpResponse, StreamingHttpResponse


def _get_microscope():
    from microscope_django.middleware import get_microscope

    return get_microscope()


def _target_path(request: HttpRequest) -> str:
    prefix = request.path
    if not prefix.startswith("/"):
        prefix = f"/{prefix}"
    return prefix


def api_view(request: HttpRequest) -> HttpResponse:
    microscope = _get_microscope()
    if not microscope.active:
        return HttpResponse(status=404)

    path = _target_path(request)
    if path.endswith("/api/stream") and request.method == "GET":
        return stream_view(request)

    result = microscope.handle(
        request.method,
        path,
        request.body.decode("utf-8", errors="replace"),
        request.META.get("QUERY_STRING", ""),
    )
    if result is None:
        return HttpResponse(status=404)

    if result.get("stream"):
        return stream_view(request)

    body = result.get("body", "")
    if isinstance(body, bytes):
        return HttpResponse(body, status=result["status"], headers=result.get("headers", {}))
    return HttpResponse(body, status=result["status"], headers=result.get("headers", {}))


def stream_view(request: HttpRequest) -> HttpResponse:
    microscope = _get_microscope()
    if not microscope.active:
        return HttpResponse(status=404)

    chunks: queue.Queue[str | None] = queue.Queue()

    def write(chunk: str) -> None:
        chunks.put(chunk)

    def run_stream() -> None:
        try:
            microscope.stream(write)
        finally:
            chunks.put(None)

    threading.Thread(target=run_stream, daemon=True).start()

    def event_stream():
        while True:
            chunk = chunks.get()
            if chunk is None:
                break
            yield chunk

    response = StreamingHttpResponse(
        streaming_content=event_stream(),
        status=200,
        content_type="text/event-stream",
    )
    response["Cache-Control"] = "no-cache, no-transform"
    response["Connection"] = "keep-alive"
    response["X-Accel-Buffering"] = "no"
    return response


def spa_view(request: HttpRequest) -> HttpResponse:
    microscope = _get_microscope()
    if not microscope.active:
        return HttpResponse(status=404)

    result = microscope.handle(request.method, _target_path(request))
    if result is None:
        return HttpResponse(status=404)

    body = result.get("body", "")
    if isinstance(body, bytes):
        return HttpResponse(body, status=result["status"], headers=result.get("headers", {}))
    return HttpResponse(body, status=result["status"], headers=result.get("headers", {}))
