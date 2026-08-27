import json

from microscope.config import Config
from microscope.detail import build_entry_detail
from microscope.entry import Entry
from microscope.hub import Hub
from microscope.http import ApiRouter
from mem_store import MemStore, wait_for_entries


def test_handler_list_and_get() -> None:
    store = MemStore()
    hub = Hub(store, Config())
    api = ApiRouter(hub)
    now = "2026-01-01T00:00:00Z"
    store.insert(Entry("e1", "b1", "request", {"method": "GET", "path": "/health", "status": 200}, now))
    store.insert(Entry("e2", "b1", "query", {"sql": "SELECT 1"}, now))

    list_result = api.handle("GET", "/microscope/api/entries")
    assert list_result["status"] == 200

    detail_result = api.handle("GET", "/microscope/api/entries/e1")
    assert detail_result["status"] == 200
    payload = json.loads(detail_result["body"])
    assert payload["entry"]["id"] == "e1"
    assert payload["content_tabs"]


def test_handler_redaction_api() -> None:
    store = MemStore()
    hub = Hub(store, Config())
    api = ApiRouter(hub)

    get_result = api.handle("GET", "/microscope/api/redaction")
    assert get_result["status"] == 200

    put_result = api.handle("PUT", "/microscope/api/redaction", json.dumps({"enabled": True}))
    assert put_result["status"] == 200
    assert hub.redact_sensitive()


def test_custom_entry_conflict_when_paused() -> None:
    store = MemStore()
    hub = Hub(store, Config())
    api = ApiRouter(hub)
    hub.set_recording_paused(True)
    result = api.handle("POST", "/microscope/api/entries", json.dumps({"name": "x"}))
    assert result["status"] == 409


def test_insights_models_validation() -> None:
    store = MemStore()
    api = ApiRouter(Hub(store, Config()))
    result = api.handle("POST", "/microscope/api/insights/models", "{}")
    assert result["status"] == 422


def test_prune_clears_all() -> None:
    store = MemStore()
    hub = Hub(store, Config())
    api = ApiRouter(hub)
    now = "2026-01-01T00:00:00Z"
    store.insert(Entry("e1", "b1", "request", {}, now))
    store.insert(Entry("e2", "b2", "query", {}, now))
    result = api.handle("POST", "/microscope/api/prune")
    assert result["status"] == 200
    assert len(store.entries) == 0
