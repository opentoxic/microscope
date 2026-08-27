from __future__ import annotations

import logging
import os
from typing import Any, Protocol

from microscope.config import Config
from microscope.db import QueryTracer
from microscope.hub import Hub
from microscope.kafka import KafkaReader, KafkaWriter
from microscope.logging_handler import MicroscopeLogHandler
from microscope.redis import RedisHook
from microscope.signals import Signals


class EventPublisher(Protocol):
    def publish(self, event_type: str, payload: dict[str, Any]) -> Any: ...


class OTPNotifier(Protocol):
    def send_signup_otp(self, email: str, otp: str) -> Any: ...
    def send_password_reset_otp(self, email: str, otp: str) -> Any: ...
    def send_email_change_otp(self, email: str, otp: str) -> Any: ...


class Integration:
    def __init__(self, app_env: str, config: Config | None = None) -> None:
        self._config = config or Config.from_env()
        self._app_env = app_env
        self._active = self._config.is_active(app_env)
        self._hub: Hub | None = None
        self._microscope: Any = None
        self._query_tracer: QueryTracer | None = QueryTracer() if self._active else None
        self._signals: Signals | None = None

    @classmethod
    def from_env(cls, app_env: str | None = None) -> Integration:
        env = app_env or os.environ.get("APP_ENV", "production")
        return cls(env)

    @property
    def active(self) -> bool:
        return self._active

    @property
    def config(self) -> Config:
        return self._config

    @property
    def hub(self) -> Hub | None:
        return self._hub

    @property
    def microscope(self) -> Any:
        return self._microscope

    @property
    def signals(self) -> Signals | None:
        return self._signals

    def query_tracer(self) -> QueryTracer | None:
        return self._query_tracer

    def bind(self, dsn: str | None = None) -> Hub | None:
        if not self._active:
            return None
        from microscope.setup import _normalize_dsn, boot

        resolved = dsn or os.environ.get("DATABASE_URL", "")
        if not resolved:
            return None
        self._microscope = boot(_normalize_dsn(resolved), self._app_env, self._config)
        if not self._microscope.active:
            self._active = False
            return None
        self._hub = self._microscope.hub
        if self._query_tracer is not None and self._hub is not None:
            self._query_tracer.bind(self._hub)
        if self._hub is not None:
            self._signals = Signals(self._hub)
        return self._hub

    def tee_logging(self, logger: logging.Logger | None = None) -> logging.Logger:
        if self._hub is None:
            return logger or logging.getLogger()
        target = logger or logging.getLogger()
        target.addHandler(MicroscopeLogHandler(self._hub))
        return target

    def redis_hook(self) -> RedisHook | None:
        if self._hub is None:
            return None
        return RedisHook(self._hub)

    def wrap_kafka_writer(self, writer: Any) -> KafkaWriter | Any:
        if self._hub is None or writer is None:
            return writer
        return KafkaWriter(writer, self._hub)

    def wrap_kafka_reader(self, reader: Any) -> KafkaReader | Any:
        if self._hub is None or reader is None:
            return reader
        return KafkaReader(reader, self._hub)

    def wrap_event_publisher(self, inner: EventPublisher) -> EventPublisher:
        if self._hub is None or inner is None:
            return inner
        hub = self._hub

        class _Wrapped:
            def publish(self, event_type: str, payload: dict[str, Any]) -> Any:
                hub.record("event", {"event_type": event_type, **(payload or {})})
                return inner.publish(event_type, payload)

        return _Wrapped()

    def wrap_otp_notifier(self, inner: OTPNotifier) -> OTPNotifier:
        if self._hub is None or inner is None:
            return inner
        hub = self._hub

        class _Wrapped:
            def send_signup_otp(self, email: str, otp: str) -> Any:
                hub.record(
                    "notification",
                    {"kind": "signup_otp", "email": email, "otp": hub.sanitize_otp(otp)},
                )
                return inner.send_signup_otp(email, otp)

            def send_password_reset_otp(self, email: str, otp: str) -> Any:
                hub.record(
                    "notification",
                    {"kind": "password_reset_otp", "email": email, "otp": hub.sanitize_otp(otp)},
                )
                return inner.send_password_reset_otp(email, otp)

            def send_email_change_otp(self, email: str, otp: str) -> Any:
                hub.record(
                    "notification",
                    {"kind": "email_change_otp", "email": email, "otp": hub.sanitize_otp(otp)},
                )
                return inner.send_email_change_otp(email, otp)

        return _Wrapped()

    def close(self) -> None:
        if self._hub is not None:
            self._hub.close()
