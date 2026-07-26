import time
import unittest
from unittest.mock import MagicMock

from microscope_client import MicroscopeClient


def _fake_response(json_body, status=200):
    resp = MagicMock()
    resp.status_code = status
    resp.json.return_value = json_body
    resp.raise_for_status.side_effect = None
    return resp


class TestRecord(unittest.TestCase):
    def setUp(self):
        self.client = MicroscopeClient(base_url="http://localhost:8093/microscope/")
        self.client.session = MagicMock()

    def test_record_posts_name_and_content(self):
        self.client.session.post.return_value = _fake_response({"id": "entry-1"})

        entry_id = self.client.record("payment_charged", content={"amount": 4200})

        self.assertEqual(entry_id, "entry-1")
        args, kwargs = self.client.session.post.call_args
        self.assertEqual(args[0], "http://localhost:8093/microscope/api/entries")
        self.assertEqual(kwargs["json"], {"name": "payment_charged", "content": {"amount": 4200}})

    def test_record_defaults_content_to_empty_dict(self):
        self.client.session.post.return_value = _fake_response({"id": "entry-2"})

        self.client.record("no_content_event")

        _, kwargs = self.client.session.post.call_args
        self.assertEqual(kwargs["json"], {"name": "no_content_event", "content": {}})

    def test_base_url_trailing_slash_is_stripped(self):
        self.assertEqual(self.client.base_url, "http://localhost:8093/microscope")


class TestListEntries(unittest.TestCase):
    def setUp(self):
        self.client = MicroscopeClient(base_url="http://localhost:8093/microscope")
        self.client.session = MagicMock()

    def test_list_entries_omits_none_params(self):
        self.client.session.get.return_value = _fake_response({"entries": [], "total": 0})

        self.client.list_entries(type="custom", limit=20)

        args, kwargs = self.client.session.get.call_args
        self.assertEqual(args[0], "http://localhost:8093/microscope/api/entries")
        self.assertEqual(kwargs["params"], {"type": "custom", "limit": 20})

    def test_list_entries_with_no_filters_sends_empty_params(self):
        self.client.session.get.return_value = _fake_response({"entries": [], "total": 0})

        self.client.list_entries()

        _, kwargs = self.client.session.get.call_args
        self.assertEqual(kwargs["params"], {})


class TestGetEntry(unittest.TestCase):
    def setUp(self):
        self.client = MicroscopeClient(base_url="http://localhost:8093/microscope")
        self.client.session = MagicMock()

    def test_get_entry_builds_correct_url(self):
        self.client.session.get.return_value = _fake_response({"id": "entry-1"})

        result = self.client.get_entry("entry-1")

        args, _ = self.client.session.get.call_args
        self.assertEqual(args[0], "http://localhost:8093/microscope/api/entries/entry-1")
        self.assertEqual(result, {"id": "entry-1"})


class TestRuntimeMetricsReporting(unittest.TestCase):
    def setUp(self):
        self.client = MicroscopeClient(base_url="http://localhost:8093/microscope")
        self.client.session = MagicMock()
        self.client.session.post.return_value = _fake_response({"id": "metric-1"})

    def tearDown(self):
        self.client.stop_runtime_metrics()

    def test_start_runtime_metrics_records_periodically(self):
        self.client.start_runtime_metrics(interval=0.05)
        deadline = time.monotonic() + 2
        while self.client.session.post.call_count < 2 and time.monotonic() < deadline:
            time.sleep(0.02)

        self.assertGreaterEqual(self.client.session.post.call_count, 2)
        _, kwargs = self.client.session.post.call_args
        self.assertEqual(kwargs["json"]["name"], "python.runtime")
        self.assertEqual(kwargs["json"]["content"]["language"], "python")

    def test_start_runtime_metrics_is_idempotent(self):
        self.client.start_runtime_metrics(interval=0.05)
        first_thread = self.client._metrics_thread
        self.client.start_runtime_metrics(interval=0.05)
        self.assertIs(self.client._metrics_thread, first_thread)

    def test_stop_runtime_metrics_halts_the_loop(self):
        self.client.start_runtime_metrics(interval=0.05)
        time.sleep(0.12)
        self.client.stop_runtime_metrics()
        count_after_stop = self.client.session.post.call_count
        time.sleep(0.15)
        self.assertEqual(self.client.session.post.call_count, count_after_stop)


if __name__ == "__main__":
    unittest.main()
