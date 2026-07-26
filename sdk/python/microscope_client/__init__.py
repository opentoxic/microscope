"""Thin HTTP client for the microscope observability API."""

from __future__ import annotations

import threading
from typing import Any

import requests

from microscope_client.runtime_metrics import sample_runtime_metrics

__all__ = ["MicroscopeClient"]


class MicroscopeClient:
    def __init__(self, base_url: str, timeout: float = 5.0) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.session = requests.Session()
        self._metrics_stop: threading.Event | None = None
        self._metrics_thread: threading.Thread | None = None

    def record(self, name: str, content: dict[str, Any] | None = None) -> str:
        resp = self.session.post(
            f"{self.base_url}/api/entries",
            json={"name": name, "content": content or {}},
            timeout=self.timeout,
        )
        resp.raise_for_status()
        return resp.json()["id"]

    def list_entries(
        self,
        type: str | None = None,
        search: str | None = None,
        limit: int | None = None,
        offset: int | None = None,
    ) -> dict[str, Any]:
        params = {
            k: v
            for k, v in {"type": type, "search": search, "limit": limit, "offset": offset}.items()
            if v is not None
        }
        resp = self.session.get(f"{self.base_url}/api/entries", params=params, timeout=self.timeout)
        resp.raise_for_status()
        return resp.json()

    def get_entry(self, entry_id: str) -> dict[str, Any]:
        resp = self.session.get(f"{self.base_url}/api/entries/{entry_id}", timeout=self.timeout)
        resp.raise_for_status()
        return resp.json()

    def start_runtime_metrics(self, interval: float = 15.0) -> None:
        """Periodically record this process's runtime metrics (threads, memory, GC).

        Safe to call once at startup; a second call is a no-op unless
        `stop_runtime_metrics` was called first.
        """
        if self._metrics_thread is not None:
            return

        stop = threading.Event()

        def loop() -> None:
            while not stop.wait(interval):
                try:
                    self.record("python.runtime", content=sample_runtime_metrics())
                except Exception:
                    pass

        thread = threading.Thread(target=loop, daemon=True, name="microscope-runtime-metrics")
        self._metrics_stop = stop
        self._metrics_thread = thread
        thread.start()

    def stop_runtime_metrics(self) -> None:
        if self._metrics_stop is not None:
            self._metrics_stop.set()
        self._metrics_thread = None
        self._metrics_stop = None
