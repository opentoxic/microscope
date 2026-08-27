from types import SimpleNamespace

from microscope.request_meta import django_request_meta


def test_django_request_meta_prefers_host_assigned_ids() -> None:
    request = SimpleNamespace(
        META={"REMOTE_ADDR": "127.0.0.1"},
        headers={},
        request_id="req_host",
        correlation_id="corr_host",
    )

    meta = django_request_meta(request)

    assert meta.request_id == "req_host"
    assert meta.correlation_id == "corr_host"


def test_django_request_meta_falls_back_to_headers() -> None:
    request = SimpleNamespace(
        META={
            "REMOTE_ADDR": "10.0.0.1",
            "HTTP_X_REQUEST_ID": "req_header",
            "HTTP_X_CORRELATION_ID": "corr_header",
        },
        headers={},
    )

    meta = django_request_meta(request)

    assert meta.request_id == "req_header"
    assert meta.correlation_id == "corr_header"


def test_django_request_meta_supports_qobly_correlation_header() -> None:
    request = SimpleNamespace(
        META={
            "REMOTE_ADDR": "10.0.0.1",
            "HTTP_X_REQUEST_ID": "req_1",
            "HTTP_X_QOBLY_CORRELATION_ID": "corr_qobly",
        },
        headers={},
    )

    meta = django_request_meta(request)

    assert meta.request_id == "req_1"
    assert meta.correlation_id == "corr_qobly"
