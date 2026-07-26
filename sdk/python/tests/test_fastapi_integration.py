import unittest
from unittest.mock import MagicMock, patch

from starlette.applications import Starlette
from starlette.responses import PlainTextResponse
from starlette.routing import Route
from starlette.testclient import TestClient

from microscope_client.integrations.fastapi import MicroscopeMiddleware


def _make_app():
    async def health(request):
        return PlainTextResponse("ok", status_code=200)

    app = Starlette(routes=[Route("/health", health)])
    app.add_middleware(MicroscopeMiddleware, base_url="http://localhost:8093/microscope")
    return app


class TestMicroscopeMiddleware(unittest.TestCase):
    @patch("microscope_client.integrations.fastapi.MicroscopeClient")
    def test_records_request_after_response(self, mock_client_cls):
        mock_client = MagicMock()
        mock_client_cls.return_value = mock_client

        app = _make_app()
        client = TestClient(app)
        response = client.get("/health")

        self.assertEqual(response.status_code, 200)
        mock_client.record.assert_called_once()
        args, kwargs = mock_client.record.call_args
        self.assertEqual(args[0], "http_request")
        content = kwargs["content"]
        self.assertEqual(content["method"], "GET")
        self.assertEqual(content["path"], "/health")
        self.assertEqual(content["status"], 200)
        self.assertIn("duration_ms", content)


if __name__ == "__main__":
    unittest.main()
