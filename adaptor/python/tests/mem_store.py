from __future__ import annotations

import json
import threading
import time
from datetime import datetime, timezone
from typing import Any

from microscope.entry import ALL_ENTRY_TYPES, Entry


class MemStore:
    def __init__(self) -> None:
        self._lock = threading.Lock()
        self.entries: list[Entry] = []
        self.settings: dict[str, bool] = {}
        self.options: dict[str, str] = {}

    def insert(self, entry: Entry) -> None:
        with self._lock:
            self.entries.append(entry)

    def get(self, entry_id: str) -> Entry | None:
        with self._lock:
            for entry in self.entries:
                if entry.id == entry_id:
                    return entry
        return None

    def list_entries(
        self,
        entry_type: str | None,
        search: str | None,
        limit: int,
        offset: int,
        request_id: str | None = None,
    ) -> dict[str, Any]:
        with self._lock:
            filtered = list(self.entries)
        if entry_type:
            filtered = [e for e in filtered if e.type == entry_type]
        if request_id:
            filtered = [e for e in filtered if e.request_id == request_id]
        if search:
            needle = search.lower()
            filtered = [
                e
                for e in filtered
                if needle in json.dumps(e.content).lower() or needle in e.request_id.lower()
            ]
        total = len(filtered)
        page = filtered[offset:offset + limit]
        return {"entries": [e.to_dict() for e in page], "total": total}

    def list_by_batch(self, batch_id: str) -> list[Entry]:
        with self._lock:
            return [e for e in self.entries if e.batch_id == batch_id]

    def prune(self, older_than: datetime) -> int:
        with self._lock:
            kept: list[Entry] = []
            deleted = 0
            for entry in self.entries:
                created = datetime.strptime(entry.created_at, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=timezone.utc)
                if created < older_than:
                    deleted += 1
                else:
                    kept.append(entry)
            self.entries = kept
            return deleted

    def clear_all(self) -> int:
        with self._lock:
            deleted = len(self.entries)
            self.entries = []
            return deleted

    def list_type_settings(self) -> list[dict[str, Any]]:
        with self._lock:
            counts: dict[str, int] = {}
            for entry in self.entries:
                counts[entry.type] = counts.get(entry.type, 0) + 1
        return [
            {
                "type": entry_type,
                "enabled": self.settings.get(entry_type, True),
                "count": counts.get(entry_type, 0),
            }
            for entry_type in ALL_ENTRY_TYPES
        ]

    def set_type_enabled(self, entry_type: str, enabled: bool) -> int:
        self.settings[entry_type] = enabled
        if not enabled:
            with self._lock:
                kept = [e for e in self.entries if e.type != entry_type]
                deleted = len(self.entries) - len(kept)
                self.entries = kept
                return deleted
        return 0

    def get_option(self, key: str) -> str | None:
        return self.options.get(key)

    def set_option(self, key: str, value: str) -> None:
        self.options[key] = value

    def storage_usage(self) -> dict[str, float | int]:
        with self._lock:
            count = len(self.entries)
        return {
            "entries_mb": float(count) * 0.01,
            "entries_data_mb": float(count) * 0.008,
            "entries_indexes_mb": float(count) * 0.002,
            "settings_mb": 0.01,
            "migrations_mb": 0.03,
            "total_mb": float(count) * 0.01 + 0.04,
            "entry_count": count,
        }


def wait_for_entries(store: MemStore, count: int, timeout: float = 2.0) -> None:
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        with store._lock:
            if len(store.entries) >= count:
                return
        time.sleep(0.01)
    raise AssertionError(f"timed out waiting for {count} entries, got {len(store.entries)}")
