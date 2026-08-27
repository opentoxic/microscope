from __future__ import annotations

import time
from typing import Any

from microscope.hub import Hub
from microscope.sanitize import truncate_bytes


class RedisHook:
    """redis-py instrumentation hook."""

    def __init__(self, hub: Hub) -> None:
        self._hub = hub

    def __call__(self, *args: Any, **kwargs: Any) -> "RedisHook":
        return self

    def dial_hook(self, next_hook):
        def hook(*args, **kwargs):
            started = time.monotonic()
            try:
                return next_hook(*args, **kwargs)
            finally:
                duration = time.monotonic() - started
                network = kwargs.get("network") or (args[1] if len(args) > 1 else "")
                address = kwargs.get("address") or (args[2] if len(args) > 2 else "")
                self._hub.record(
                    "redis",
                    {
                        "command": "DIAL",
                        "duration_ms": duration * 1000,
                        "network": network,
                        "address": address,
                    },
                    tags=["redis:DIAL"],
                )

        return hook

    def process_hook(self, next_hook):
        def hook(*args, **kwargs):
            command = kwargs.get("command") or (args[1] if len(args) > 1 else None)
            started = time.monotonic()
            error = None
            try:
                return next_hook(*args, **kwargs)
            except Exception as exc:
                error = exc
                raise
            finally:
                duration = time.monotonic() - started
                content: dict[str, Any] = {"argument_count": 0}
                command_name = "UNKNOWN"
                if command is not None:
                    command_name = str(getattr(command, "name", command)).upper()
                    args_list = list(getattr(command, "args", []) or [])
                    content["argument_count"] = max(0, len(args_list) - 1)
                    if not self._hub.redact_sensitive() and len(args_list) > 1:
                        content["args"] = args_list[1:]
                if error is not None:
                    content["error"] = str(error)
                self._hub.record(
                    "redis",
                    {
                        "command": command_name,
                        "duration_ms": duration * 1000,
                        **content,
                    },
                    tags=[f"redis:{command_name}"],
                )

        return hook

    def process_pipeline_hook(self, next_hook):
        def hook(*args, **kwargs):
            commands = kwargs.get("commands") or (args[1] if len(args) > 1 else [])
            started = time.monotonic()
            error = None
            try:
                return next_hook(*args, **kwargs)
            except Exception as exc:
                error = exc
                raise
            finally:
                duration = time.monotonic() - started
                names = [str(getattr(cmd, "name", cmd)).upper() for cmd in commands]
                content: dict[str, Any] = {"commands": names, "count": len(commands)}
                if not self._hub.redact_sensitive():
                    args_list = []
                    for command in commands:
                        cmd_args = list(getattr(command, "args", []) or [])
                        args_list.append(cmd_args[1:] if len(cmd_args) > 1 else None)
                    content["args"] = args_list
                if error is not None:
                    content["error"] = str(error)
                self._hub.record(
                    "redis",
                    {
                        "command": "PIPELINE",
                        "duration_ms": duration * 1000,
                        **content,
                    },
                    tags=["redis:PIPELINE"],
                )

        return hook
