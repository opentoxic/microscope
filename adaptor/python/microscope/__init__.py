"""Native Python adaptor for microscope."""

from microscope.setup import boot
from microscope.hub import Hub

__all__ = ["boot", "Hub"]
