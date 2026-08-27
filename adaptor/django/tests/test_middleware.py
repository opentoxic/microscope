import json
from unittest.mock import patch

import django
from django.conf import settings
from django.http import HttpResponse, StreamingHttpResponse
from django.test import RequestFactory

if not settings.configured:
    settings.configure(
        DEBUG=True,
        SECRET_KEY="test",
        ROOT_URLCONF=__name__,
        MIDDLEWARE=[],
        MICROSCOPE_PATH="/microscope",
    )
    django.setup()

from microscope.config import Config
from microscope.detail import build_entry_detail
from microscope.hub import Hub
from microscope.http import ApiRouter, SpaRouter
from microscope.setup import Microscope
from microscope_django.middleware import RecordRequestsMiddleware
from mem_store import MemStore, wait_for_entries


def _boot_test_microscope(store: MemStore) -> Microscope:
    hub = Hub(store, Config())
    return Microscope(
        hub=hub,
        api=ApiRouter(hub),
        spa=SpaRouter(__import__("pathlib").Path("/tmp"), "/microscope"),
        active=True,
    )


def test_middleware_skips_microscope_paths() -> None:
    store = MemStore()
    microscope = _boot_test_microscope(store)

    with patch("microscope_django.middleware.get_microscope", return_value=microscope):
        middleware = RecordRequestsMiddleware(lambda req: HttpResponse("ok"))
        factory = RequestFactory()

        microscope_req = factory.get("/microscope/")
        middleware(microscope_req)

        health_req = factory.get("/health", HTTP_X_REQUEST_ID="request-test")
        middleware(health_req)
        wait_for_entries(store, 1)

    assert len(store.entries) == 1
    assert store.entries[0].type == "request"
    assert store.entries[0].content["path"] == "/health"
    assert store.entries[0].request_id == "request-test"


def test_middleware_records_rich_request_fields() -> None:
    store = MemStore()
    microscope = _boot_test_microscope(store)

    with patch("microscope_django.middleware.get_microscope", return_value=microscope):
        middleware = RecordRequestsMiddleware(
            lambda req: HttpResponse(json.dumps({"ok": True}), content_type="application/json")
        )
        factory = RequestFactory()
        request = factory.post("/api/users", data=b'{"email":"a@b.com"}', content_type="application/json")
        middleware(request)
        wait_for_entries(store, 1)

    content = store.entries[0].content
    assert content["method"] == "POST"
    assert "request_body" in content
    assert "headers" in content
    assert '{"email":"a@b.com"}' in content["request_body"]
    assert '"ok": true' in content["response_body"]


def test_middleware_bridges_host_request_and_correlation_ids() -> None:
    store = MemStore()
    microscope = _boot_test_microscope(store)

    def view(request):
        return HttpResponse('{"error":"unauthenticated"}', status=401, content_type="application/json")

    with patch("microscope_django.middleware.get_microscope", return_value=microscope):
        middleware = RecordRequestsMiddleware(view)
        factory = RequestFactory()
        request = factory.post(
            "/v1/workspaces",
            data=b'{"name":"","website_url":null}',
            content_type="application/json",
        )
        request.request_id = "req_host_abc"
        request.correlation_id = "corr_host_xyz"
        middleware(request)
        wait_for_entries(store, 1)

    entry = store.entries[0]
    assert entry.request_id == "req_host_abc"
    assert entry.correlation_id == "corr_host_xyz"
    assert entry.content["status"] == 401
    assert "website_url" in entry.content["request_body"]
    assert "unauthenticated" in entry.content["response_body"]


def test_middleware_captures_streaming_response_body() -> None:
    store = MemStore()
    microscope = _boot_test_microscope(store)

    def view(_request):
        response = StreamingHttpResponse(iter([b'{"streamed":true}']), content_type="application/json")
        response.status_code = 200
        return response

    with patch("microscope_django.middleware.get_microscope", return_value=microscope):
        middleware = RecordRequestsMiddleware(view)
        request = RequestFactory().get("/stream")
        middleware(request)
        wait_for_entries(store, 1)

    assert store.entries[0].content["response_body"] == ""


def test_detail_api_includes_payload_and_response_tabs() -> None:
    store = MemStore()
    microscope = _boot_test_microscope(store)

    with patch("microscope_django.middleware.get_microscope", return_value=microscope):
        middleware = RecordRequestsMiddleware(
            lambda req: HttpResponse(
                json.dumps({"error": {"code": "unauthenticated"}}),
                status=401,
                content_type="application/json",
            )
        )
        request = RequestFactory().post(
            "/v1/workspaces",
            data=b'{"name":"demo"}',
            content_type="application/json",
        )
        request.request_id = "req_detail"
        request.correlation_id = "corr_detail"
        middleware(request)
        wait_for_entries(store, 1)

    entry = store.entries[0]
    detail = build_entry_detail(entry, store.list_by_batch(entry.batch_id))
    tab_ids = {tab["id"] for tab in detail["content_tabs"]}

    assert "payload" in tab_ids
    assert "headers" in tab_ids
    assert "response" in tab_ids
    assert detail["entry"]["request_id"] == "req_detail"
