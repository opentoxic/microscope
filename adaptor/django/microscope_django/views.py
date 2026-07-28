from __future__ import annotations

from django.http import HttpRequest, HttpResponse, StreamingHttpResponse

from microscope_django.middleware import get_microscope


def _target_path(request: HttpRequest) -> str:
    prefix = request.path
    if not prefix.startswith("/"):
        prefix = f"/{prefix}"
    return prefix


def api_view(request: HttpRequest) -> HttpResponse:
    microscope = get_microscope()
    if not microscope.active:
        return HttpResponse(status=404)

    result = microscope.handle(
        request.method,
        _target_path(request),
        request.body.decode("utf-8"),
        request.META.get("QUERY_STRING", ""),
    )
    if result is None:
        return HttpResponse(status=404)

    body = result.get("body", "")
    if isinstance(body, bytes):
        return HttpResponse(body, status=result["status"], headers=result.get("headers", {}))
    return HttpResponse(body, status=result["status"], headers=result.get("headers", {}))


def stream_view(request: HttpRequest) -> HttpResponse:
    microscope = get_microscope()
    if not microscope.active:
        return HttpResponse(status=404)

    def event_stream():
        yield ""

    return StreamingHttpResponse(
        event_stream(),
        status=200,
        content_type="text/event-stream",
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )


def spa_view(request: HttpRequest) -> HttpResponse:
    microscope = get_microscope()
    if not microscope.active:
        return HttpResponse(status=404)

    result = microscope.handle(request.method, _target_path(request))
    if result is None:
        return HttpResponse(status=404)

    body = result.get("body", "")
    if isinstance(body, bytes):
        return HttpResponse(body, status=result["status"], headers=result.get("headers", {}))
    return HttpResponse(body, status=result["status"], headers=result.get("headers", {}))
