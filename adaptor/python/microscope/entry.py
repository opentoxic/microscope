from __future__ import annotations

from dataclasses import dataclass, field
from typing import Any

ALL_ENTRY_TYPES = [
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


@dataclass
class Entry:
    id: str
    batch_id: str
    type: str
    content: dict[str, Any]
    created_at: str
    request_id: str = ""
    correlation_id: str = ""
    tags: list[str] = field(default_factory=list)

    def to_dict(self) -> dict[str, Any]:
        data: dict[str, Any] = {
            "id": self.id,
            "batch_id": self.batch_id,
            "type": self.type,
            "content": self.content,
            "created_at": self.created_at,
        }
        if self.request_id:
            data["request_id"] = self.request_id
        if self.correlation_id:
            data["correlation_id"] = self.correlation_id
        if self.tags:
            data["tags"] = self.tags
        return data
