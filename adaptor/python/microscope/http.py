from __future__ import annotations

import json
import mimetypes
from pathlib import Path
from typing import Any
from urllib.parse import parse_qs

from microscope.hub import Hub


class ApiRouter:
    def __init__(self, hub: Hub) -> None:
        self._hub = hub

    def handle(self, method: str, path: str, body: str = "", query: str = "") -> dict[str, Any]:
        prefix = self._hub.config.path_prefix().rstrip("/")
        params = parse_qs(query.lstrip("?"), keep_blank_values=True)

        if path == f"{prefix}/api/entries" and method == "GET":
            return self._json(
                200,
                self._hub.store.list_entries(
                    params.get("type", [None])[0],
                    params.get("search", [None])[0],
                    int(params.get("limit", ["50"])[0]),
                    int(params.get("offset", ["0"])[0]),
                ),
            )

        if path == f"{prefix}/api/entries" and method == "POST":
            return self._create_custom(body)

        if path.startswith(f"{prefix}/api/entries/") and method == "GET":
            entry_id = path.rsplit("/", 1)[-1]
            return self._get_entry(entry_id)

        if path == f"{prefix}/api/stream" and method == "GET":
            return {"status": 200, "headers": {"Content-Type": "text/event-stream"}, "body": "", "stream": True}

        if path == f"{prefix}/api/prune" and method == "POST":
            deleted = self._hub.store.clear_all()
            return self._json(200, {"deleted": deleted})

        if path == f"{prefix}/api/storage" and method == "GET":
            return self._json(200, self._hub.store.storage_usage())

        if path == f"{prefix}/api/recording" and method == "GET":
            return self._json(200, {"paused": self._hub.recording_paused()})

        if path == f"{prefix}/api/recording" and method == "PUT":
            return self._set_recording(body)

        if path == f"{prefix}/api/redaction" and method == "GET":
            return self._json(200, {"enabled": self._hub.redact_sensitive()})

        if path == f"{prefix}/api/redaction" and method == "PUT":
            return self._set_redaction(body)

        if path == f"{prefix}/api/settings" and method == "GET":
            return self._json(200, {"settings": self._hub.type_settings()})

        if path.startswith(f"{prefix}/api/settings/") and method == "PUT":
            entry_type = path.rsplit("/", 1)[-1]
            return self._update_setting(entry_type, body)

        return self._json(404, {"error": "not found"})

    def _create_custom(self, body: str) -> dict[str, Any]:
        payload = json.loads(body or "{}")
        name = payload.get("name")
        if not isinstance(name, str) or not name.strip():
            return self._json(400, {"error": "name is required"})
        entry_id = self._hub.record("custom", {"name": name, **(payload.get("content") or {})})
        return self._json(202, {"id": entry_id})

    def _get_entry(self, entry_id: str) -> dict[str, Any]:
        entry = self._hub.store.get(entry_id)
        if entry is None:
            return self._json(404, {"error": "not found"})
        batch = [item.to_dict() for item in self._hub.store.list_by_batch(entry.batch_id)]
        return self._json(200, {"entry": entry.to_dict(), "batch": batch})

    def _set_recording(self, body: str) -> dict[str, Any]:
        payload = json.loads(body or "{}")
        paused = bool(payload.get("paused"))
        self._hub.set_recording_paused(paused)
        return self._json(200, {"paused": paused})

    def _set_redaction(self, body: str) -> dict[str, Any]:
        payload = json.loads(body or "{}")
        enabled = bool(payload.get("enabled"))
        self._hub.set_redact_sensitive(enabled)
        return self._json(200, {"enabled": enabled})

    def _update_setting(self, entry_type: str, body: str) -> dict[str, Any]:
        payload = json.loads(body or "{}")
        enabled = bool(payload.get("enabled"))
        deleted = self._hub.set_type_enabled(entry_type, enabled)
        return self._json(200, {"type": entry_type, "enabled": enabled, "deleted": deleted})

    @staticmethod
    def _json(status: int, payload: dict[str, Any]) -> dict[str, Any]:
        return {
            "status": status,
            "headers": {"Content-Type": "application/json"},
            "body": json.dumps(payload),
        }


class SpaRouter:
    def __init__(self, dist_path: Path, path_prefix: str) -> None:
        self._dist_path = dist_path
        self._path_prefix = path_prefix.rstrip("/")

    def handle(self, method: str, path: str) -> dict[str, Any] | None:
        if method != "GET" or not path.startswith(self._path_prefix):
            return None

        relative = path[len(self._path_prefix) :].lstrip("/")
        if relative.startswith("api/"):
            return None

        if relative in {"", "settings"} or relative.startswith("entries/"):
            return self._serve_file("index.html", "text/html; charset=utf-8")

        if relative.startswith("assets/"):
            mime = mimetypes.guess_type(relative)[0] or "application/octet-stream"
            return self._serve_file(relative, mime)

        return None

    def _serve_file(self, relative: str, content_type: str) -> dict[str, Any]:
        file_path = self._dist_path / relative
        if not file_path.is_file():
            return {"status": 404, "headers": {"Content-Type": "text/plain"}, "body": "not found"}
        return {
            "status": 200,
            "headers": {"Content-Type": content_type},
            "body": file_path.read_text(encoding="utf-8") if content_type.startswith("text/") else file_path.read_bytes(),
        }
