from __future__ import annotations

import copy
from typing import Any

from microscope.redact import redact_args, redact_headers, redact_json, redact_map


def deep_clone_map(mapping: dict[str, Any] | None) -> dict[str, Any] | None:
    if mapping is None:
        return None
    return copy.deepcopy(mapping)


def sanitize_map(mapping: dict[str, Any] | None, redact_sensitive: bool) -> dict[str, Any] | None:
    if mapping is None:
        return None
    if redact_sensitive:
        return redact_map(mapping)
    return deep_clone_map(mapping)


def sanitize_headers(headers: dict[str, list[str]] | None, redact_sensitive: bool) -> dict[str, list[str]] | None:
    if headers is None:
        return None
    if redact_sensitive:
        return redact_headers(headers)
    out: dict[str, list[str]] = {}
    for key, values in headers.items():
        out[key] = list(values)
    return out


def sanitize_json(body: bytes, redact_sensitive: bool) -> str:
    if not body:
        return ""
    if redact_sensitive:
        return redact_json(body)
    return body.decode("utf-8", errors="replace")


def sanitize_args(args: list[Any], redact_sensitive: bool) -> list[Any] | None:
    if not args:
        return None
    if redact_sensitive:
        return redact_args(args)
    out: list[Any] = []
    for arg in args:
        if isinstance(arg, dict):
            out.append(sanitize_map(arg, redact_sensitive=False))
        elif isinstance(arg, (bytes, bytearray)):
            out.append(bytes(arg).decode("utf-8", errors="replace"))
        else:
            out.append(arg)
    return out


def truncate_bytes(data: bytes, max_len: int) -> str:
    if max_len <= 0 or not data:
        return ""
    if len(data) <= max_len:
        return data.decode("utf-8", errors="replace")
    return data[:max_len].decode("utf-8", errors="replace")
