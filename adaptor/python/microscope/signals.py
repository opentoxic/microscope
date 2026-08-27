from __future__ import annotations

import json
import threading
from datetime import datetime, timezone
from typing import Any

from microscope.context import batch_id_from_context
from microscope.entry import Entry
from microscope.hub import Hub
from microscope.request_meta import request_meta_from_context


def _milliseconds(duration_seconds: float) -> float:
    return duration_seconds * 1000


def _clone_content(content: dict[str, Any] | None) -> dict[str, Any]:
    cloned: dict[str, Any] = {}
    if content:
        cloned.update(content)
    return cloned


class Signals:
    def __init__(self, hub: Hub) -> None:
        self._hub = hub

    def record_cache(
        self,
        operation: str,
        key: str,
        hit: bool,
        duration_seconds: float,
        content: dict[str, Any] | None = None,
    ) -> None:
        payload = _clone_content(content)
        payload.update(
            {
                "operation": operation,
                "key": key,
                "hit": hit,
                "duration_ms": _milliseconds(duration_seconds),
            }
        )
        self._record_typed("cache", [f"cache:{operation}"], payload)

    def record_redis(
        self,
        command: str,
        duration_seconds: float,
        error: Exception | None = None,
        content: dict[str, Any] | None = None,
    ) -> None:
        payload = _clone_content(content)
        payload.update({"command": command, "duration_ms": _milliseconds(duration_seconds)})
        if error is not None:
            payload["error"] = str(error)
        self._record_typed("redis", [f"redis:{command}"], payload)

    def record_job(
        self,
        name: str,
        queue: str,
        state: str,
        duration_seconds: float,
        content: dict[str, Any] | None = None,
    ) -> None:
        payload = _clone_content(content)
        payload.update(
            {
                "name": name,
                "queue": queue,
                "state": state,
                "duration_ms": _milliseconds(duration_seconds),
            }
        )
        self._record_typed("job", [f"job:{state}", f"queue:{queue}"], payload)

    def record_schedule(
        self,
        name: str,
        state: str,
        duration_seconds: float,
        content: dict[str, Any] | None = None,
    ) -> None:
        payload = _clone_content(content)
        payload.update(
            {
                "name": name,
                "state": state,
                "duration_ms": _milliseconds(duration_seconds),
            }
        )
        self._record_typed("schedule", [f"schedule:{state}"], payload)

    def record_mail(
        self,
        subject: str,
        recipients: list[str],
        state: str,
        duration_seconds: float,
        content: dict[str, Any] | None = None,
    ) -> None:
        payload = _clone_content(content)
        payload.update(
            {
                "subject": subject,
                "recipients": recipients,
                "state": state,
                "duration_ms": _milliseconds(duration_seconds),
            }
        )
        self._record_typed("mail", [f"mail:{state}"], payload)

    def record_websocket(
        self,
        event: str,
        channel: str,
        direction: str,
        size: int,
        content: dict[str, Any] | None = None,
    ) -> None:
        payload = _clone_content(content)
        payload.update(
            {
                "event": event,
                "channel": channel,
                "direction": direction,
                "size_bytes": size,
            }
        )
        self._record_typed("websocket", [f"websocket:{event}"], payload)

    def record_performance(
        self,
        name: str,
        duration_seconds: float,
        content: dict[str, Any] | None = None,
    ) -> None:
        payload = _clone_content(content)
        payload.update({"name": name, "duration_ms": _milliseconds(duration_seconds)})
        self._record_typed("performance", [f"performance:{name}"], payload)

    def record_metric(
        self,
        name: str,
        value: float,
        unit: str,
        content: dict[str, Any] | None = None,
    ) -> None:
        payload = _clone_content(content)
        payload.update({"name": name, "value": value, "unit": unit})
        self._record_typed("metric", [f"metric:{name}"], payload)

    def record_custom(self, name: str, content: dict[str, Any] | None = None) -> None:
        payload = _clone_content(content)
        payload["name"] = name
        self._record_typed("custom", [f"custom:{name}"], payload)

    def record_topic(
        self,
        topic: str,
        action: str,
        duration_seconds: float,
        content: dict[str, Any] | None = None,
    ) -> None:
        payload = _clone_content(content)
        payload.update(
            {
                "topic": topic,
                "action": action,
                "duration_ms": _milliseconds(duration_seconds),
            }
        )
        self._record_typed("topic", [f"topic:{topic}", f"kafka:{action}"], payload)

    def record_event(self, event_type: str, payload: dict[str, Any] | None = None) -> None:
        content = _clone_content(payload)
        content["event_type"] = event_type
        self._record_typed("event", [f"event:{event_type}"], content)

    def record_notification(self, kind: str, content: dict[str, Any] | None = None) -> None:
        payload = _clone_content(content)
        payload["kind"] = kind
        self._record_typed("notification", [f"notification:{kind}"], payload)

    def timed(self, name: str, content: dict[str, Any] | None = None):
        import time

        started = time.monotonic()
        payload = _clone_content(content)

        def finish(error: Exception | None = None) -> None:
            if error is not None:
                payload["error"] = str(error)
            self.record_performance(name, time.monotonic() - started, payload)

        return finish

    def _record_typed(self, entry_type: str, tags: list[str], content: dict[str, Any]) -> None:
        meta = request_meta_from_context()
        entry = Entry(
            id="",
            batch_id=batch_id_from_context(),
            type=entry_type,
            content=self._hub.sanitize_map(content) or {},
            created_at=datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
            request_id=meta.request_id,
            correlation_id=meta.correlation_id,
            tags=tags,
        )
        self._hub.record_entry(entry)


def attach_signals(hub: Hub) -> Signals:
    return Signals(hub)
