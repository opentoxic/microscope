from __future__ import annotations

import json
import logging
import queue
import secrets
import threading
import time
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timedelta, timezone
from typing import Any, Callable

from microscope.config import Config
from microscope.context import batch_id_from_context, without_trace
from microscope.entry import ALL_ENTRY_TYPES, Entry
from microscope.request_meta import request_meta_from_context
from microscope.runtime_metrics import sample
from microscope.sanitize import sanitize_args, sanitize_headers, sanitize_json, sanitize_map
from microscope.store import PostgresStore

_log = logging.getLogger("microscope")


class Hub:
    def __init__(self, store: PostgresStore, config: Config) -> None:
        self._store = store
        self._config = config
        self._enabled = {entry_type: True for entry_type in ALL_ENTRY_TYPES}
        self._recording_paused = False
        self._redact_sensitive = config.redact_sensitive
        self._subs_lock = threading.RLock()
        self._entry_subscribers: list[queue.Queue[Entry]] = []
        self._control_subscribers: list[queue.Queue[dict[str, Any]]] = []
        self._stop = threading.Event()
        self._executor = ThreadPoolExecutor(max_workers=4, thread_name_prefix="microscope")
        self._background_threads: list[threading.Thread] = []
        self._load_type_settings()
        self._load_redaction_setting()
        self._start_pruner()
        self._start_runtime_sampler()

    @property
    def config(self) -> Config:
        return self._config

    @property
    def store(self) -> PostgresStore:
        return self._store

    def record(
        self,
        entry_type: str,
        content: dict[str, Any],
        batch_id: str | None = None,
        request_id: str | None = None,
        entry_id: str | None = None,
        tags: list[str] | None = None,
        correlation_id: str | None = None,
    ) -> str:
        if self._recording_paused or not self.type_enabled(entry_type):
            return ""

        meta = request_meta_from_context()
        resolved_batch = batch_id or batch_id_from_context()
        resolved_request_id = request_id or meta.request_id
        resolved_correlation = correlation_id or meta.correlation_id

        entry = Entry(
            id=entry_id or self._new_id(),
            batch_id=resolved_batch or entry_id or self._new_id(),
            type=entry_type,
            content=self.sanitize_map(content),
            created_at=self._utc_now(),
            request_id=resolved_request_id,
            correlation_id=resolved_correlation,
            tags=tags or [],
        )
        if not resolved_batch:
            entry.batch_id = entry.id
        self._submit_entry(entry)
        return entry.id

    def record_entry(self, entry: Entry) -> None:
        if self._recording_paused or not self.type_enabled(entry.type):
            return
        if not entry.id:
            entry.id = self._new_id()
        if not entry.batch_id:
            entry.batch_id = batch_id_from_context() or entry.id
        entry.content = self.sanitize_map(entry.content) or {}
        self._submit_entry(entry)

    def _submit_entry(self, entry: Entry) -> None:
        self._executor.submit(self._insert_and_publish, entry)

    def _insert_and_publish(self, entry: Entry) -> None:
        if self._recording_paused or not self.type_enabled(entry.type):
            return
        try:
            without_trace()
            self._store.insert(entry)
            self._publish(entry)
        except Exception:
            _log.debug("microscope: failed to persist entry", exc_info=True)

    def type_enabled(self, entry_type: str) -> bool:
        return entry_type in ALL_ENTRY_TYPES and self._enabled.get(entry_type, False)

    def recording_paused(self) -> bool:
        return self._recording_paused

    def set_recording_paused(self, paused: bool) -> None:
        if self._recording_paused == paused:
            return
        self._recording_paused = paused
        self._publish_control({"action": "recording-paused", "paused": paused})

    def redact_sensitive(self) -> bool:
        return self._redact_sensitive

    def set_redact_sensitive(self, enabled: bool) -> None:
        if self._redact_sensitive == enabled:
            return
        self._redact_sensitive = enabled
        self._store.set_option("redact_sensitive", json.dumps(enabled))
        self._publish_control({"action": "redaction-setting", "redact_sensitive": enabled})

    def type_settings(self) -> list[dict[str, Any]]:
        return self._store.list_type_settings()

    def set_type_enabled(self, entry_type: str, enabled: bool) -> int:
        if entry_type not in ALL_ENTRY_TYPES:
            raise ValueError(f"unknown signal type: {entry_type}")
        previous = self._enabled.get(entry_type, True)
        self._enabled[entry_type] = enabled
        try:
            deleted = self._store.set_type_enabled(entry_type, enabled)
        except Exception:
            self._enabled[entry_type] = previous
            raise
        self._publish_control({"action": "signal-setting", "type": entry_type, "deleted": deleted})
        return deleted

    def clear_all(self) -> int:
        deleted = self._store.clear_all()
        self._publish_control({"action": "clear-all", "deleted": deleted})
        return deleted

    def prune(self) -> int:
        cutoff = datetime.now(timezone.utc) - self._config.retention_timedelta()
        return self._store.prune(cutoff)

    def subscribe(self, buffer: int = 64) -> tuple[queue.Queue[Entry], Callable[[], None]]:
        if buffer <= 0:
            buffer = 32
        ch: queue.Queue[Entry] = queue.Queue(maxsize=buffer)
        with self._subs_lock:
            self._entry_subscribers.append(ch)

        def unsubscribe() -> None:
            with self._subs_lock:
                if ch in self._entry_subscribers:
                    self._entry_subscribers.remove(ch)

        return ch, unsubscribe

    def subscribe_control(self, buffer: int = 8) -> tuple[queue.Queue[dict[str, Any]], Callable[[], None]]:
        if buffer <= 0:
            buffer = 8
        ch: queue.Queue[dict[str, Any]] = queue.Queue(maxsize=buffer)
        with self._subs_lock:
            self._control_subscribers.append(ch)

        def unsubscribe() -> None:
            with self._subs_lock:
                if ch in self._control_subscribers:
                    self._control_subscribers.remove(ch)

        return ch, unsubscribe

    def subscribe_legacy(self, callback: Callable[[Entry], None]) -> None:
        entries_q, _ = self.subscribe()
        threading.Thread(target=self._fan_out_entries, args=(entries_q, callback), daemon=True).start()

    def subscribe_control_legacy(self, callback: Callable[[dict[str, Any]], None]) -> None:
        controls_q, _ = self.subscribe_control()
        threading.Thread(target=self._fan_out_controls, args=(controls_q, callback), daemon=True).start()

    def _fan_out_entries(self, ch: queue.Queue[Entry], callback: Callable[[Entry], None]) -> None:
        while not self._stop.is_set():
            try:
                callback(ch.get(timeout=1))
            except queue.Empty:
                continue

    def _fan_out_controls(self, ch: queue.Queue[dict[str, Any]], callback: Callable[[dict[str, Any]], None]) -> None:
        while not self._stop.is_set():
            try:
                callback(ch.get(timeout=1))
            except queue.Empty:
                continue

    def sanitize_map(self, mapping: dict[str, Any] | None) -> dict[str, Any] | None:
        return sanitize_map(mapping, self._redact_sensitive)

    def sanitize_headers(self, headers: dict[str, list[str]] | None) -> dict[str, list[str]] | None:
        return sanitize_headers(headers, self._redact_sensitive)

    def sanitize_json(self, body: bytes) -> str:
        return sanitize_json(body, self._redact_sensitive)

    def sanitize_args(self, args: list[Any]) -> list[Any] | None:
        return sanitize_args(args, self._redact_sensitive)

    def sanitize_otp(self, otp: str) -> str:
        if self._redact_sensitive:
            return "[REDACTED]"
        return otp

    def close(self) -> None:
        self._stop.set()
        self._executor.shutdown(wait=True, cancel_futures=True)
        with self._subs_lock:
            self._entry_subscribers.clear()
            self._control_subscribers.clear()

    def _publish(self, entry: Entry) -> None:
        with self._subs_lock:
            for subscriber in self._entry_subscribers:
                try:
                    subscriber.put_nowait(entry)
                except queue.Full:
                    pass

    def _publish_control(self, event: dict[str, Any]) -> None:
        with self._subs_lock:
            for subscriber in self._control_subscribers:
                try:
                    subscriber.put_nowait(event)
                except queue.Full:
                    pass

    def _load_type_settings(self) -> None:
        for setting in self._store.list_type_settings():
            self._enabled[setting["type"]] = setting["enabled"]

    def _load_redaction_setting(self) -> None:
        raw = self._store.get_option("redact_sensitive")
        if raw is None:
            return
        decoded = json.loads(raw)
        if isinstance(decoded, bool):
            self._redact_sensitive = decoded

    def _start_pruner(self) -> None:
        thread = threading.Thread(target=self._pruner_loop, name="microscope-pruner", daemon=True)
        thread.start()
        self._background_threads.append(thread)

    def _pruner_loop(self) -> None:
        while not self._stop.wait(timeout=3600):
            try:
                self.prune()
            except Exception:
                _log.debug("microscope: retention prune failed", exc_info=True)

    def _start_runtime_sampler(self) -> None:
        thread = threading.Thread(target=self._runtime_sampler_loop, name="microscope-runtime", daemon=True)
        thread.start()
        self._background_threads.append(thread)

    def _runtime_sampler_loop(self) -> None:
        while not self._stop.wait(timeout=15):
            if self._recording_paused or not self.type_enabled("metric"):
                continue
            entry_id = self._new_id()
            entry = Entry(
                id=entry_id,
                batch_id=entry_id,
                type="metric",
                tags=["metric:python.runtime"],
                content=sample(),
                created_at=self._utc_now(),
            )
            self._submit_entry(entry)

    @staticmethod
    def _new_id() -> str:
        return secrets.token_hex(16)

    @staticmethod
    def _utc_now() -> str:
        return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
