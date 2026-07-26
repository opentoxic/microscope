import unittest
from unittest.mock import MagicMock

from django.conf import settings
from django.http import HttpResponse
from django.test import RequestFactory

if not settings.configured:
    settings.configure(MICROSCOPE_BASE_URL="http://localhost:8093/microscope")

from microscope_client.integrations.django import MicroscopeMiddleware


class TestMicroscopeMiddleware(unittest.TestCase):
    def setUp(self):
        self.factory = RequestFactory()
        self.get_response = MagicMock(return_value=HttpResponse(status=204))
        self.middleware = MicroscopeMiddleware(self.get_response)
        self.middleware.client = MagicMock()

    def test_records_request_after_response(self):
        request = self.factory.get("/health")

        response = self.middleware(request)

        self.assertEqual(response.status_code, 204)
        self.middleware.client.record.assert_called_once()
        name, kwargs = self.middleware.client.record.call_args
        self.assertEqual(name[0], "http_request")
        content = kwargs["content"]
        self.assertEqual(content["method"], "GET")
        self.assertEqual(content["path"], "/health")
        self.assertEqual(content["status"], 204)
        self.assertIn("duration_ms", content)

    def test_calls_get_response(self):
        request = self.factory.post("/auth/login")

        self.middleware(request)

        self.get_response.assert_called_once_with(request)


if __name__ == "__main__":
    unittest.main()
