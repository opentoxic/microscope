import json
from unittest.mock import patch

import django
from django.conf import settings
from django.http import HttpResponse
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

from microscope_django.middleware import RecordRequestsMiddleware
from microscope.config import Config
from microscope.hub import Hub
from microscope.http import ApiRouter, SpaRouter
from microscope.setup import Microscope
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
