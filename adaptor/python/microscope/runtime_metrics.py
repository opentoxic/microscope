from __future__ import annotations

import gc
import sys
import threading
from typing import Any

try:
    import resource

    _HAS_RESOURCE = True
except ImportError:  # pragma: no cover - POSIX only
    _HAS_RESOURCE = False


def _memory_mb() -> float:
    if not _HAS_RESOURCE:
        return 0.0
    return resource.getrusage(resource.RUSAGE_SELF).ru_maxrss / 1024


def sample() -> dict[str, Any]:
    stats = gc.get_stats()
    gc_collections = sum(item.get("collections", 0) for item in stats)

    return {
        "name": "python.runtime",
        "language": "python",
        "value": threading.active_count(),
        "unit": "threads",
        "threads": threading.active_count(),
        "memory_mb": _memory_mb(),
        "gc_objects": len(gc.get_objects(0)) if stats else 0,
        "gc_collections": gc_collections,
        "python_version": sys.version.split()[0],
    }
