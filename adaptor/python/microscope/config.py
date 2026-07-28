from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass
class Config:
    enabled: bool = True
    path: str = "/microscope"
    retention_hours: int = 24
    max_body_bytes: int = 65536
    allowed_envs: tuple[str, ...] = ("development", "local")
    auto_migrate: bool = True
    redact_sensitive: bool = False

    @classmethod
    def from_env(cls) -> "Config":
        return cls(
            enabled=os.getenv("MICROSCOPE_ENABLED", "true").lower() in {"1", "true", "yes"},
            path=os.getenv("MICROSCOPE_PATH", "/microscope"),
            retention_hours=int(os.getenv("MICROSCOPE_RETENTION_HOURS", "24")),
            max_body_bytes=int(os.getenv("MICROSCOPE_MAX_BODY_BYTES", "65536")),
            allowed_envs=tuple(
                e.strip()
                for e in os.getenv("MICROSCOPE_ALLOWED_ENVS", "development,local").split(",")
                if e.strip()
            ),
            auto_migrate=os.getenv("MICROSCOPE_AUTO_MIGRATE", "true").lower() in {"1", "true", "yes"},
            redact_sensitive=os.getenv("MICROSCOPE_REDACT_SENSITIVE", "false").lower() in {"1", "true", "yes"},
        )

    def path_prefix(self) -> str:
        path = self.path.strip("/")
        return f"/{path}" if path else "/microscope"

    def is_active(self, app_env: str) -> bool:
        return self.enabled and app_env in self.allowed_envs
