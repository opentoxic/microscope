from __future__ import annotations

import json
import secrets
from collections.abc import Callable
from typing import Any

from microscope.config import Config
from microscope.entry import ALL_ENTRY_TYPES, Entry
from microscope.store import PostgresStore


class Hub:
    def __init__(self, store: PostgresStore, config: Config) -> None:
        self._store = store
        self._config = config
        self._enabled = {entry_type: True for entry_type in ALL_ENTRY_TYPES}
        self._recording_paused = False
        self._redact_sensitive = config.redact_sensitive
        self._subscribers: list[Callable[[Entry], None]] = []
        self._control_subscribers: list[Callable[[dict[str, Any]], None]] = []
        self._load_type_settings()
        self._load_redaction_setting()

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
    ) -> str:
        if self._recording_paused or not self.type_enabled(entry_type):
            return ""

        entry_id = self._new_id()
        entry = Entry(
            id=entry_id,
            batch_id=batch_id or entry_id,
            type=entry_type,
            content=content,
            created_at=self._utc_now(),
            request_id=request_id or "",
        )
        self._store.insert(entry)
        self._publish(entry)
        return entry_id

    def record_entry(self, entry: Entry) -> None:
        if self._recording_paused or not self.type_enabled(entry.type):
            return
        if not entry.id:
            entry.id = self._new_id()
        if not entry.batch_id:
            entry.batch_id = entry.id
        self._store.insert(entry)
        self._publish(entry)

    def type_enabled(self, entry_type: str) -> bool:
        return entry_type in ALL_ENTRY_TYPES and self._enabled.get(entry_type, False)

    def recording_paused(self) -> bool:
        return self._recording_paused

    def set_recording_paused(self, paused: bool) -> None:
        self._recording_paused = paused
        self._publish_control({"action": "recording-paused", "paused": paused})

    def redact_sensitive(self) -> bool:
        return self._redact_sensitive

    def set_redact_sensitive(self, enabled: bool) -> None:
        self._redact_sensitive = enabled
        self._store.set_option("redact_sensitive", json.dumps(enabled))
        self._publish_control({"action": "redaction", "redact_sensitive": enabled})

    def type_settings(self) -> list[dict[str, Any]]:
        return self._store.list_type_settings()

    def set_type_enabled(self, entry_type: str, enabled: bool) -> int:
        if entry_type not in ALL_ENTRY_TYPES:
            raise ValueError(f"unknown signal type: {entry_type}")
        self._enabled[entry_type] = enabled
        deleted = self._store.set_type_enabled(entry_type, enabled)
        self._publish_control({"action": "signal-setting", "type": entry_type, "deleted": deleted})
        return deleted

    def subscribe(self, callback: Callable[[Entry], None]) -> None:
        self._subscribers.append(callback)

    def subscribe_control(self, callback: Callable[[dict[str, Any]], None]) -> None:
        self._control_subscribers.append(callback)

    def _publish(self, entry: Entry) -> None:
        for callback in self._subscribers:
            callback(entry)

    def _publish_control(self, event: dict[str, Any]) -> None:
        for callback in self._control_subscribers:
            callback(event)

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

    @staticmethod
    def _new_id() -> str:
        return secrets.token_hex(16)

    @staticmethod
    def _utc_now() -> str:
        from datetime import datetime, timezone

        return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
