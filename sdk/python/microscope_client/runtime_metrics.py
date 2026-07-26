"""Best-effort Python runtime metrics, shaped like every other microscope SDK.

Every SDK reports the same envelope so the dashboard can render any of them
generically: name, language, a primary concurrency value + unit, plus a few
language-specific extras.
"""

from __future__ import annotations

import gc
import threading
from typing import Any

try:
    import resource

    _HAS_RESOURCE = True
except ImportError:  # pragma: no cover - resource is POSIX-only
    _HAS_RESOURCE = False


def _memory_mb() -> float:
    if not _HAS_RESOURCE:
        return 0.0
    # ru_maxrss is KB on Linux, bytes on macOS; KB is the common case and
    # this value is best-effort, not exact.
    return resource.getrusage(resource.RUSAGE_SELF).ru_maxrss / 1024


def sample_runtime_metrics() -> dict[str, Any]:
    """Return the current process's runtime metrics as microscope entry content."""
    stats = gc.get_stats()
    gc_collections = sum(s.get("collections", 0) for s in stats)

    return {
        "name": "python.runtime",
        "language": "python",
        "value": threading.active_count(),
        "unit": "threads",
        "threads": threading.active_count(),
        "memory_mb": _memory_mb(),
        "gc_objects": len(gc.get_objects(0)) if stats else 0,
        "gc_collections": gc_collections,
    }
