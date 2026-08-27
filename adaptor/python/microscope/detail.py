from __future__ import annotations

import json
from typing import Any

from microscope.batch_group import BatchTypeGroup, group_batch_by_type
from microscope.content import content_float, content_string, json_pretty
from microscope.entry import Entry


class ContentTab:
    def __init__(self, id: str, label: str, body: str, json_tab: bool = False) -> None:
        self.id = id
        self.label = label
        self.body = body
        self.json = json_tab

    def to_dict(self) -> dict[str, Any]:
        return {"id": self.id, "label": self.label, "body": self.body, "json": self.json}


def build_entry_detail(entry: Entry, batch: list[Entry]) -> dict[str, Any]:
    batch_groups = group_batch_related(batch, entry.id)
    return {
        "entry": entry.to_dict(),
        "batch": [item.to_dict() for item in batch],
        "batch_groups": [group.to_dict() for group in batch_groups],
        "content_tabs": [tab.to_dict() for tab in entry_content_tabs(entry)],
        "related_active_tab": first_related_tab_type(batch_groups),
    }


def entry_content_tabs(entry: Entry) -> list[ContentTab]:
    content = entry.content or {}
    tabs: list[ContentTab] = []

    if entry.type == "request":
        request_body = content_string(content.get("request_body"))
        if request_body and request_body not in {"null", "{}"}:
            tabs.append(
                ContentTab("payload", "Payload", pretty_content(request_body), looks_like_json(request_body))
            )
        headers = content.get("headers")
        if headers is not None:
            tabs.append(ContentTab("headers", "Headers", json_pretty_any(headers), True))
        response_body = content_string(content.get("response_body"))
        if response_body and response_body != "null":
            tabs.append(
                ContentTab("response", "Response", pretty_content(response_body), looks_like_json(response_body))
            )
    elif entry.type == "query":
        sql = content_string(content.get("sql"))
        if sql:
            tabs.append(ContentTab("query", "Query", sql))
    elif entry.type == "exception":
        stack = content_string(content.get("stack"))
        if stack:
            tabs.append(ContentTab("stack", "Stack Trace", stack))
    elif entry.type == "topic":
        tabs.append(ContentTab("message", "Message metadata", json_pretty(content), True))
    elif entry.type not in {
        "log",
        "event",
        "notification",
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
    }:
        tabs.append(ContentTab("payload", "Payload", json_pretty(content), True))

    if not tabs:
        tabs.append(ContentTab("payload", "Payload", json_pretty(content), True))
    return tabs


def group_batch_related(batch: list[Entry], current_id: str) -> list[BatchTypeGroup]:
    filtered = [entry for entry in batch if entry.id != current_id]
    return group_batch_by_type(filtered)


def first_related_tab_type(groups: list[BatchTypeGroup]) -> str:
    if not groups:
        return ""
    return groups[0].type


def batch_group_summary(group: BatchTypeGroup) -> str:
    count = len(group.entries)
    if group.type == "query":
        dup = count_duplicate_queries(group.entries)
        word = "query" if count == 1 else "queries"
        return f"{count} {word}, {dup} of which are duplicated."
    if group.type == "log":
        word = "log" if count == 1 else "logs"
        return f"{count} {word}"
    if group.type == "request":
        word = "request" if count == 1 else "requests"
        return f"{count} {word}"
    word = group.label.lower()
    return f"{count} {word}"


def batch_group_total_duration(group: BatchTypeGroup) -> str:
    if group.type != "query":
        return ""
    total = sum(content_float(entry.content.get("duration_ms")) for entry in group.entries)
    if total < 1:
        return f"{total:.2f}ms"
    return f"{total:.0f}ms"


def count_duplicate_queries(entries: list[Entry]) -> int:
    seen: dict[str, int] = {}
    dup = 0
    for entry in entries:
        sql = content_string(entry.content.get("sql"))
        seen[sql] = seen.get(sql, 0) + 1
        if seen[sql] == 2:
            dup += 1
    return dup


def pretty_content(value: str) -> str:
    if looks_like_json(value):
        return pretty_json_string(value)
    return value


def looks_like_json(value: str) -> bool:
    stripped = value.strip()
    if not stripped:
        return False
    try:
        json.loads(stripped)
        return True
    except json.JSONDecodeError:
        return False


def pretty_json_string(value: str) -> str:
    stripped = value.strip()
    try:
        parsed = json.loads(stripped)
        return json.dumps(parsed, indent=2)
    except json.JSONDecodeError:
        return value


def json_pretty_any(value: Any) -> str:
    if isinstance(value, dict):
        if all(isinstance(v, list) for v in value.values()):
            normalized = {k: v for k, v in value.items()}
            return json_pretty(normalized)
        return json_pretty(value)
    try:
        return json.dumps(value, indent=2)
    except (TypeError, ValueError):
        text = content_string(value)
        return text if text else "{}"

