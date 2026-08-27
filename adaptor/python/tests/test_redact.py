from microscope.config import Config
from microscope.hub import Hub
from microscope.redact import redact_headers, redact_json, redact_map
from microscope.sanitize import sanitize_map
from mem_store import MemStore


def test_redact_map() -> None:
    mapping = {
        "email": "user@example.com",
        "password": "secret123",
        "otp": "123456",
        "nested": {"refresh_token": "abc", "name": "test"},
    }
    out = redact_map(mapping)
    assert out["password"] == "[REDACTED]"
    assert out["otp"] == "[REDACTED]"
    assert out["email"] == "user@example.com"
    assert out["nested"]["refresh_token"] == "[REDACTED]"


def test_redact_headers() -> None:
    headers = {"Authorization": ["Bearer secret"], "Content-Type": ["application/json"]}
    out = redact_headers(headers)
    assert out["Authorization"] == ["[REDACTED]"]
    assert out["Content-Type"] == ["application/json"]


def test_redact_json() -> None:
    body = b'{"email":"a@b.com","password":"pw"}'
    out = redact_json(body)
    assert "pw" not in out


def test_hub_sanitize_modes() -> None:
    store = MemStore()
    hub = Hub(store, Config(redact_sensitive=False))
    full = hub.sanitize_map({"password": "secret", "email": "user@example.com"})
    assert full["password"] == "secret"

    redacting = Hub(store, Config(redact_sensitive=True))
    redacted = redacting.sanitize_map({"password": "secret"})
    assert redacted["password"] == "[REDACTED]"


def test_hub_sanitize_otp() -> None:
    store = MemStore()
    hub = Hub(store, Config(redact_sensitive=False))
    assert hub.sanitize_otp("123456") == "123456"
    redacting = Hub(store, Config(redact_sensitive=True))
    assert redacting.sanitize_otp("123456") == "[REDACTED]"
