import queue
import threading
import time
from unittest.mock import patch

import django
from django.conf import settings
from django.test import RequestFactory

if not settings.configured:
    settings.configure(DEBUG=True, SECRET_KEY="test", ROOT_URLCONF=__name__)
    django.setup()

from microscope.config import Config
from microscope.hub import Hub
from microscope.http import ApiRouter
from microscope.setup import Microscope
from microscope_django.views import stream_view
from mem_store import MemStore, wait_for_entries


def test_stream_view_emits_connected_and_entry() -> None:
    store = MemStore()
    hub = Hub(store, Config())
    microscope = Microscope(hub=hub, api=ApiRouter(hub), spa=None, active=True)

    factory = RequestFactory()
    request = factory.get("/microscope/api/stream")

    with patch("microscope_django.middleware.get_microscope", return_value=microscope):
        response = stream_view(request)

    chunks: queue.Queue[str | None] = queue.Queue()

    def consume() -> None:
        for chunk in response.streaming_content:
            chunks.put(chunk.decode("utf-8"))

    thread = threading.Thread(target=consume, daemon=True)
    thread.start()

    connected = chunks.get(timeout=2)
    assert ": connected" in connected

    hub.record("custom", {"name": "stream-test"})
    wait_for_entries(store, 1)

    deadline = time.monotonic() + 2
    found = False
    while time.monotonic() < deadline:
        try:
            chunk = chunks.get(timeout=0.1)
            if "stream-test" in chunk:
                found = True
                break
        except queue.Empty:
            continue
    assert found
