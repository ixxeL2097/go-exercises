"""
Package contenant les routes de l'API v1.
"""
from .deployments import router as deployments_router
from .health import router as health_router

__all__ = ["deployments_router", "health_router"]