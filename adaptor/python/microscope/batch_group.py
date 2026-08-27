from __future__ import annotations

from dataclasses import dataclass
from typing import Any

from microscope.entry import Entry

BATCH_TYPE_ORDER = [
    "request",
    "query",
    "log",
    "event",
    "notification",
    "exception",
    "cache",
    "redis",
    "job",
    "schedule",
    "mail",
    "http-client",
    "websocket",
    "performance",
    "metric",
    "custom",
    "topic",
]

TYPE_LABELS = {
    "request": "Requests",
    "query": "Queries",
    "log": "Logs",
    "event": "Events",
    "notification": "Notifications",
    "exception": "Exceptions",
    "cache": "Cache",
    "redis": "Redis",
    "job": "Queue Jobs",
    "schedule": "Scheduled Tasks",
    "mail": "Mail",
    "http-client": "External Calls",
    "websocket": "WebSockets",
    "performance": "Performance",
    "metric": "Metrics",
    "custom": "Custom Events",
    "topic": "Redpanda Topics",
}


@dataclass
class BatchTypeGroup:
    type: str
    label: str
    entries: list[Entry]

    def to_dict(self) -> dict[str, Any]:
        return {
            "type": self.type,
            "label": self.label,
            "entries": [entry.to_dict() for entry in self.entries],
        }


def type_label(entry_type: str) -> str:
    return TYPE_LABELS.get(entry_type, entry_type)


def group_batch_by_type(batch: list[Entry]) -> list[BatchTypeGroup]:
    by_type: dict[str, list[Entry]] = {t: [] for t in BATCH_TYPE_ORDER}
    for entry in batch:
        by_type.setdefault(entry.type, []).append(entry)
    groups: list[BatchTypeGroup] = []
    for entry_type in BATCH_TYPE_ORDER:
        entries = by_type.get(entry_type, [])
        if not entries:
            continue
        groups.append(BatchTypeGroup(type=entry_type, label=type_label(entry_type), entries=entries))
    return groups
