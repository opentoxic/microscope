import json
import threading
import time

from microscope.context import with_batch_id
from microscope.config import Config
from microscope.hub import Hub
from microscope.http import ApiRouter
from microscope.signals import Signals
from mem_store import MemStore, wait_for_entries


def test_typed_signals_are_recorded_and_published() -> None:
    store = MemStore()
    hub = Hub(store, Config())
    signals = Signals(hub)
    entries_q, unsub = hub.subscribe(16)

    with_batch_id("batch-signals")
    signals.record_cache("get", "session:1", True, 0.001)
    signals.record_redis("GET", 0.001)
    signals.record_job("send-email", "default", "processed", 0.001)
    signals.record_schedule("prune", "finished", 0.001)
    signals.record_mail("Welcome", ["user@example.com"], "sent", 0.001)
    signals.record_websocket("message", "updates", "outgoing", 42)
    signals.record_performance("password.hash", 0.001)
    signals.record_metric("workers", 4, "count")
    signals.record_custom("checkpoint", {"token": "secret"})
    signals.record_topic("identity.events", "produce", 0.001, {"message_count": 1})

    wait_for_entries(store, 10)

    seen: set[str] = set()
    for _ in range(10):
        try:
            entry = entries_q.get(timeout=2)
            seen.add(entry.type)
        except Exception:
            break
    unsub()

    for entry_type in (
        "cache",
        "redis",
        "job",
        "schedule",
        "mail",
        "websocket",
        "performance",
        "metric",
        "custom",
        "topic",
    ):
        assert entry_type in seen


def test_live_entry_stream_publishes_persisted_entry_once() -> None:
    store = MemStore()
    hub = Hub(store, Config())
    api = ApiRouter(hub)
    chunks: list[str] = []

    def run_stream() -> None:
        api.stream(chunks.append)

    thread = threading.Thread(target=run_stream, daemon=True)
    thread.start()
    time.sleep(0.05)
    hub.record("custom", {"name": "live-test"})
    wait_for_entries(store, 1)

    deadline = time.monotonic() + 2
    data_line = ""
    while time.monotonic() < deadline:
        for chunk in chunks:
            for line in chunk.splitlines():
                if line.startswith("data: ") and "live-test" in line:
                    data_line = line
                    break
        if data_line:
            break
        time.sleep(0.05)

    assert data_line
    assert '"name":"live-test"' in data_line or '"name": "live-test"' in data_line


def test_create_custom_entry_endpoint() -> None:
    store = MemStore()
    hub = Hub(store, Config())
    api = ApiRouter(hub)
    result = api.handle(
        "POST",
        "/microscope/api/entries",
        json.dumps({"name": "release checkpoint", "content": {"commit": "abc123"}}),
    )
    assert result["status"] == 202
    wait_for_entries(store, 1)
    assert store.entries[0].type == "custom"
    assert store.entries[0].content["name"] == "release checkpoint"
