import asyncio

from microscope.asgi import MicroscopeASGIMiddleware
from microscope.setup import Microscope


class _StubHub:
    def __init__(self) -> None:
        self.config = type("Cfg", (), {"path_prefix": staticmethod(lambda: "/microscope"), "max_body_bytes": 65536})()
        self.recorded: list[tuple[str, dict]] = []

    def record(self, entry_type: str, content: dict, **kwargs) -> str:
        self.recorded.append((entry_type, content))
        return "id"

    def sanitize_headers(self, headers):
        return headers

    def sanitize_json(self, body):
        return body.decode("utf-8", errors="replace") if isinstance(body, bytes) else str(body)


def _run(coro):
    return asyncio.run(coro)


def _make_scope(path: str, method: str = "GET") -> dict:
    return {"type": "http", "path": path, "method": method, "query_string": b"", "headers": []}


async def _receive() -> dict:
    return {"type": "http.request", "body": b"", "more_body": False}


def _collect_send() -> tuple[list, callable]:
    messages: list = []

    async def send(message: dict) -> None:
        messages.append(message)

    return messages, send


async def _app(scope, receive, send) -> None:
    await send({"type": "http.response.start", "status": 200, "headers": []})
    await send({"type": "http.response.body", "body": b"ok"})


def test_passthrough_request_is_recorded_and_forwarded() -> None:
    hub = _StubHub()
    microscope = Microscope(hub=hub, api=None, spa=None, active=True)
    middleware = MicroscopeASGIMiddleware(_app, microscope)

    messages, send = _collect_send()
    _run(middleware(_make_scope("/users"), _receive, send))

    assert messages[0]["status"] == 200
    assert messages[1]["body"] == b"ok"
    assert hub.recorded[0][0] == "request"
    assert hub.recorded[0][1]["path"] == "/users"
    assert hub.recorded[0][1]["status"] == 200


def test_inactive_microscope_skips_recording_and_still_forwards() -> None:
    microscope = Microscope(hub=None, api=None, spa=None, active=False)
    middleware = MicroscopeASGIMiddleware(_app, microscope)

    messages, send = _collect_send()
    _run(middleware(_make_scope("/users"), _receive, send))

    assert messages[0]["status"] == 200


def test_microscope_own_path_is_handled_without_hitting_downstream_app() -> None:
    hub = _StubHub()
    microscope = Microscope(hub=hub, api=None, spa=None, active=True)
    microscope.handle = lambda method, path, body="", query="": {  # type: ignore[method-assign]
        "status": 200,
        "headers": {"Content-Type": "application/json"},
        "body": "{}",
    }

    async def unreachable_app(scope, receive, send) -> None:
        raise AssertionError("downstream app should not be called for microscope's own path")

    middleware = MicroscopeASGIMiddleware(unreachable_app, microscope)

    messages, send = _collect_send()
    _run(middleware(_make_scope("/microscope/api/entries"), _receive, send))

    assert messages[0]["status"] == 200
    assert messages[1]["body"] == b"{}"
