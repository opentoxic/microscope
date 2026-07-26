"""Thin HTTP client for the microscope observability API."""

from __future__ import annotations

from typing import Any

import requests

__all__ = ["MicroscopeClient"]


class MicroscopeClient:
    def __init__(self, base_url: str, timeout: float = 5.0) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.session = requests.Session()

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
