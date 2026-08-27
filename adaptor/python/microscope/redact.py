from __future__ import annotations

import json
from typing import Any

SENSITIVE_KEYS = frozenset(
    {
        "password",
        "password_hash",
        "new_password",
        "old_password",
        "current_password",
        "refresh_token",
        "access_token",
        "token",
        "otp",
        "code",
        "secret",
        "encryption_key",
        "authorization",
        "mfa_secret",
        "backup_codes",
    }
)

SENSITIVE_HEADER_KEYS = frozenset(
    {
        "authorization",
        "cookie",
        "x-api-key",
        "x-auth-token",
    }
)


def redact_map(mapping: dict[str, Any] | None) -> dict[str, Any] | None:
    if mapping is None:
        return None
    return {key: _redact_value(key, value) for key, value in mapping.items()}


def redact_headers(headers: dict[str, list[str]] | None) -> dict[str, list[str]] | None:
    if headers is None:
        return None
    out: dict[str, list[str]] = {}
    for key, values in headers.items():
        if key.lower() in SENSITIVE_HEADER_KEYS:
            out[key] = ["[REDACTED]"]
        else:
            out[key] = list(values)
    return out


def redact_json(body: bytes) -> str:
    if not body:
        return ""
    try:
        data = json.loads(body)
    except json.JSONDecodeError:
        return body.decode("utf-8", errors="replace")
    redacted = _redact_value("", data)
    try:
        return json.dumps(redacted)
    except (TypeError, ValueError):
        return body.decode("utf-8", errors="replace")


def redact_args(args: list[Any]) -> list[Any] | None:
    if not args:
        return None
    out: list[Any] = []
    for arg in args:
        if isinstance(arg, dict):
            out.append(redact_map(arg))
        elif isinstance(arg, (bytes, bytearray)):
            out.append("[bytes]")
        else:
            out.append(arg)
    return out


def _redact_value(key: str, value: Any) -> Any:
    if isinstance(key, str) and key.lower() in SENSITIVE_KEYS:
        return "[REDACTED]"
    if isinstance(value, dict):
        return redact_map(value)
    if isinstance(value, list):
        return [_redact_value(key, item) for item in value]
    return value
