from __future__ import annotations

import json
from typing import Any


def content_string(value: Any) -> str:
    if value is None:
        return ""
    if isinstance(value, str):
        return value
    try:
        return json.dumps(value)
    except (TypeError, ValueError):
        return ""


def content_float(value: Any) -> float:
    if isinstance(value, bool):
        return float(value)
    if isinstance(value, (int, float)):
        return float(value)
    if isinstance(value, str):
        try:
            return float(value)
        except ValueError:
            return 0.0
    return 0.0


def json_pretty(value: dict[str, Any] | None) -> str:
    if value is None:
        return "{}"
    try:
        return json.dumps(value, indent=2)
    except (TypeError, ValueError):
        return "{}"
