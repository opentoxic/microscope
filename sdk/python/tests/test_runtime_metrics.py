import unittest

from microscope_client.runtime_metrics import sample_runtime_metrics


class TestSampleRuntimeMetrics(unittest.TestCase):
    def test_returns_expected_shape(self):
        metrics = sample_runtime_metrics()

        self.assertEqual(metrics["name"], "python.runtime")
        self.assertEqual(metrics["language"], "python")
        self.assertEqual(metrics["unit"], "threads")
        self.assertIsInstance(metrics["value"], int)
        self.assertGreaterEqual(metrics["value"], 1)
        self.assertIsInstance(metrics["memory_mb"], float)
        self.assertGreaterEqual(metrics["memory_mb"], 0.0)
        self.assertIsInstance(metrics["gc_objects"], int)
        self.assertIsInstance(metrics["gc_collections"], int)

    def test_thread_count_reflects_active_threads(self):
        import threading

        started = threading.Event()
        stop = threading.Event()

        def worker():
            started.set()
            stop.wait(2)

        thread = threading.Thread(target=worker, daemon=True)
        thread.start()
        started.wait(1)
        try:
            metrics = sample_runtime_metrics()
            self.assertGreaterEqual(metrics["value"], 2)
        finally:
            stop.set()
            thread.join(timeout=1)


if __name__ == "__main__":
    unittest.main()
