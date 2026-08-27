import os

from microscope.config import Config
from microscope.integration import Integration


def test_config_from_env() -> None:
    os.environ["MICROSCOPE_ENABLED"] = "true"
    os.environ["MICROSCOPE_PATH"] = "/signals"
    os.environ["MICROSCOPE_RETENTION_HOURS"] = "48"
    os.environ["MICROSCOPE_MAX_BODY_BYTES"] = "8192"
    os.environ["MICROSCOPE_ALLOWED_ENVS"] = "development,staging"
    os.environ["MICROSCOPE_AUTO_MIGRATE"] = "false"

    cfg = Config.from_env()
    assert cfg.enabled
    assert cfg.path == "/signals"
    assert cfg.retention_hours == 48
    assert cfg.max_body_bytes == 8192
    assert cfg.allowed_envs == ("development", "staging")
    assert not cfg.auto_migrate


def test_integration_inactive() -> None:
    integration = Integration("development", Config(enabled=False))
    assert not integration.active
    assert integration.query_tracer() is None
    assert integration.bind("postgres://invalid") is None


def test_integration_active_without_pool() -> None:
    integration = Integration("development", Config())
    assert integration.active
    assert integration.query_tracer() is not None
