from __future__ import annotations

import time
from typing import Any

from microscope.hub import Hub
from microscope.sanitize import truncate_bytes


class KafkaWriter:
    def __init__(self, writer: Any, hub: Hub) -> None:
        self._writer = writer
        self._hub = hub

    def write_messages(self, *args, **kwargs) -> Any:
        messages = kwargs.get("messages")
        if messages is None and args:
            messages = args[0]
        messages = list(messages or [])
        started = time.monotonic()
        error = None
        try:
            return self._writer.write_messages(*args, **kwargs)
        except Exception as exc:
            error = exc
            raise
        finally:
            topic = getattr(self._writer, "topic", "") or ""
            if not topic and messages:
                topic = getattr(messages[0], "topic", "") or messages[0].get("topic", "")
            size_bytes = 0
            for message in messages:
                key = getattr(message, "key", None) or message.get("key", b"")
                value = getattr(message, "value", None) or message.get("value", b"")
                size_bytes += len(key or b"") + len(value or b"")
            content: dict[str, Any] = {
                "message_count": len(messages),
                "size_bytes": size_bytes,
                "error": str(error) if error else "",
            }
            if not self._hub.redact_sensitive():
                max_bytes = self._hub.config.max_body_bytes or 65536
                payloads = []
                for message in messages:
                    key = getattr(message, "key", None) or message.get("key", b"")
                    value = getattr(message, "value", None) or message.get("value", b"")
                    payloads.append(
                        {
                            "key": truncate_bytes(bytes(key or b""), max_bytes),
                            "value": truncate_bytes(bytes(value or b""), max_bytes),
                        }
                    )
                content["messages"] = payloads
            self._hub.record(
                "topic",
                {
                    "topic": topic,
                    "action": "produce",
                    "duration_ms": (time.monotonic() - started) * 1000,
                    **content,
                },
                tags=[f"topic:{topic}", "kafka:produce"],
            )

    def close(self) -> Any:
        return self._writer.close()


class KafkaReader:
    def __init__(self, reader: Any, hub: Hub) -> None:
        self._reader = reader
        self._hub = hub

    def fetch_message(self, *args, **kwargs) -> Any:
        started = time.monotonic()
        error = None
        message = None
        try:
            message = self._reader.fetch_message(*args, **kwargs)
            return message
        except Exception as exc:
            error = exc
            raise
        finally:
            topic = ""
            partition = 0
            offset = 0
            key = b""
            value = b""
            if message is not None:
                topic = getattr(message, "topic", "") or message.get("topic", "")
                partition = getattr(message, "partition", 0) or message.get("partition", 0)
                offset = getattr(message, "offset", 0) or message.get("offset", 0)
                key = getattr(message, "key", None) or message.get("key", b"")
                value = getattr(message, "value", None) or message.get("value", b"")
            if not topic:
                topic = getattr(self._reader, "topic", "") or getattr(self._reader, "_topic", "")
            content: dict[str, Any] = {
                "partition": partition,
                "offset": offset,
                "size_bytes": len(key or b"") + len(value or b""),
                "error": str(error) if error else "",
            }
            if not self._hub.redact_sensitive():
                max_bytes = self._hub.config.max_body_bytes or 65536
                content["key"] = truncate_bytes(bytes(key or b""), max_bytes)
                content["value"] = truncate_bytes(bytes(value or b""), max_bytes)
            self._hub.record(
                "topic",
                {
                    "topic": topic,
                    "action": "consume",
                    "duration_ms": (time.monotonic() - started) * 1000,
                    **content,
                },
                tags=[f"topic:{topic}", "kafka:consume"],
            )

    def commit_messages(self, *args, **kwargs) -> Any:
        messages = kwargs.get("messages")
        if messages is None and args:
            messages = args[0]
        messages = list(messages or [])
        started = time.monotonic()
        error = None
        try:
            return self._reader.commit_messages(*args, **kwargs)
        except Exception as exc:
            error = exc
            raise
        finally:
            topic = getattr(self._reader, "topic", "") or getattr(self._reader, "_topic", "")
            if not topic and messages:
                message = messages[0]
                topic = getattr(message, "topic", "") or message.get("topic", "")
            self._hub.record(
                "topic",
                {
                    "topic": topic,
                    "action": "commit",
                    "duration_ms": (time.monotonic() - started) * 1000,
                    "message_count": len(messages),
                    "error": str(error) if error else "",
                },
                tags=[f"topic:{topic}", "kafka:commit"],
            )

    def close(self) -> Any:
        return self._reader.close()
