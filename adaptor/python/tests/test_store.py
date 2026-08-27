from datetime import datetime, timedelta, timezone

from microscope.config import Config
from microscope.entry import Entry
from microscope.hub import Hub
from mem_store import MemStore, wait_for_entries


def test_clear_all_commits_before_vacuum_semantics() -> None:
    store = MemStore()
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    store.insert(Entry("e1", "b1", "request", {}, now))
    deleted = store.clear_all()
    assert deleted == 1
    assert len(store.entries) == 0


def test_prune_removes_old_entries() -> None:
    store = MemStore()
    old = "2020-01-01T00:00:00Z"
    new = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    store.insert(Entry("e1", "b1", "request", {}, old))
    store.insert(Entry("e2", "b2", "request", {}, new))
    cutoff = datetime.now(timezone.utc) - timedelta(hours=1)
    deleted = store.prune(cutoff)
    assert deleted == 1
    assert len(store.entries) == 1
    assert store.entries[0].id == "e2"


def test_hub_clear_all_publishes_control() -> None:
    store = MemStore()
    hub = Hub(store, Config())
    controls_q, unsub = hub.subscribe_control(4)
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    store.insert(Entry("e1", "b1", "request", {}, now))
    hub.clear_all()
    event = controls_q.get(timeout=2)
    unsub()
    assert event["action"] == "clear-all"
    assert event["deleted"] == 1


def test_list_entries_request_id_filter() -> None:
    store = MemStore()
    now = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    store.insert(Entry("e1", "b1", "request", {}, now, request_id="req-1"))
    store.insert(Entry("e2", "b2", "request", {}, now, request_id="req-2"))
    result = store.list_entries(None, None, 50, 0, "req-1")
    assert result["total"] == 1
    assert result["entries"][0]["id"] == "e1"
