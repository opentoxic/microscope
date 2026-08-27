from __future__ import annotations

import json
import mimetypes
import queue
import time
from pathlib import Path
from typing import Any, Callable
from urllib.parse import parse_qs

from microscope.detail import build_entry_detail
from microscope.entry import ALL_ENTRY_TYPES
from microscope.hub import Hub
from microscope.insights import run_insight_analysis
from microscope.llm_models import list_provider_models


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
                    params.get("request_id", [None])[0],
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
            deleted = self._hub.clear_all()
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

        if path == f"{prefix}/api/insights/analyze" and method == "POST":
            return self._analyze_insights(body)

        if path == f"{prefix}/api/insights/models" and method == "POST":
            return self._list_models(body)

        return self._json(404, {"error": "not found"})

    def stream(self, write: Callable[[str], None]) -> None:
        write(": connected\n\n")
        entries_q, unsub_entries = self._hub.subscribe(64)
        controls_q, unsub_controls = self._hub.subscribe_control(8)
        try:
            heartbeat_at = time.monotonic() + 20
            while True:
                try:
                    control = controls_q.get_nowait()
                    payload = json.dumps(control)
                    write(f"event: control\ndata: {payload}\n\n")
                except queue.Empty:
                    pass

                timeout = max(0.1, heartbeat_at - time.monotonic())
                try:
                    entry = entries_q.get(timeout=timeout)
                    payload = json.dumps(entry.to_dict())
                    write(f"id: {entry.id}\nevent: entry\ndata: {payload}\n\n")
                    heartbeat_at = time.monotonic() + 20
                except queue.Empty:
                    write(": heartbeat\n\n")
                    heartbeat_at = time.monotonic() + 20
        finally:
            unsub_entries()
            unsub_controls()

    def _create_custom(self, body: str) -> dict[str, Any]:
        if self._hub.recording_paused():
            return self._json(409, {"error": "recording is paused"})
        if not self._hub.type_enabled("custom"):
            return self._json(409, {"error": "custom events are disabled in settings"})
        try:
            payload = json.loads(body or "{}")
        except json.JSONDecodeError:
            return self._json(400, {"error": "invalid JSON body"})
        name = str(payload.get("name", "")).strip()
        if not name or len(name) > 120:
            return self._json(422, {"error": "name must contain 1 to 120 characters"})
        content = payload.get("content") if isinstance(payload.get("content"), dict) else {}
        content = dict(content)
        content["name"] = name
        entry_id = self._hub.record("custom", content, tags=["custom:" + name])
        return self._json(202, {"id": entry_id})

    def _get_entry(self, entry_id: str) -> dict[str, Any]:
        entry = self._hub.store.get(entry_id)
        if entry is None:
            return self._json(404, {"error": "entry not found"})
        batch = self._hub.store.list_by_batch(entry.batch_id)
        return self._json(200, build_entry_detail(entry, batch))

    def _set_recording(self, body: str) -> dict[str, Any]:
        try:
            payload = json.loads(body or "{}")
        except json.JSONDecodeError:
            return self._json(400, {"error": "paused must be a boolean"})
        paused = payload.get("paused")
        if not isinstance(paused, bool):
            return self._json(400, {"error": "paused must be a boolean"})
        self._hub.set_recording_paused(paused)
        return self._json(200, {"paused": self._hub.recording_paused()})

    def _set_redaction(self, body: str) -> dict[str, Any]:
        try:
            payload = json.loads(body or "{}")
        except json.JSONDecodeError:
            return self._json(400, {"error": "enabled must be a boolean"})
        enabled = payload.get("enabled")
        if not isinstance(enabled, bool):
            return self._json(400, {"error": "enabled must be a boolean"})
        self._hub.set_redact_sensitive(enabled)
        return self._json(200, {"enabled": self._hub.redact_sensitive()})

    def _update_setting(self, entry_type: str, body: str) -> dict[str, Any]:
        if entry_type not in ALL_ENTRY_TYPES:
            return self._json(404, {"error": "unknown signal type"})
        try:
            payload = json.loads(body or "{}")
        except json.JSONDecodeError:
            return self._json(400, {"error": "enabled must be a boolean"})
        enabled = payload.get("enabled")
        if not isinstance(enabled, bool):
            return self._json(400, {"error": "enabled must be a boolean"})
        deleted = self._hub.set_type_enabled(entry_type, enabled)
        return self._json(200, {"type": entry_type, "enabled": enabled, "deleted": deleted})

    def _analyze_insights(self, body: str) -> dict[str, Any]:
        try:
            payload = json.loads(body or "{}")
        except json.JSONDecodeError:
            return self._json(400, {"error": "invalid JSON body"})
        provider = str(payload.get("provider", "")).strip().lower()
        model = str(payload.get("model", "")).strip()
        api_key = str(payload.get("api_key", "")).strip()
        entries = payload.get("entries") or []
        if not provider or not model or not api_key:
            return self._json(422, {"error": "provider, model, and api_key are required"})
        if not entries:
            return self._json(422, {"error": "at least one entry is required"})
        if len(entries) > 120:
            payload["entries"] = entries[:120]
        try:
            result = run_insight_analysis(payload)
        except Exception as exc:
            return self._json(502, {"error": str(exc)})
        return self._json(200, result)

    def _list_models(self, body: str) -> dict[str, Any]:
        try:
            payload = json.loads(body or "{}")
        except json.JSONDecodeError:
            return self._json(400, {"error": "invalid JSON body"})
        provider = str(payload.get("provider", "")).strip().lower()
        api_key = str(payload.get("api_key", "")).strip()
        if not provider or not api_key:
            return self._json(422, {"error": "provider and api_key are required"})
        try:
            models = list_provider_models(provider, api_key)
        except Exception as exc:
            return self._json(502, {"error": str(exc)})
        return self._json(200, {"models": models})

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
