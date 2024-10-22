"""
Routes de l'API v1.
"""
from fastapi import APIRouter
from .routes.deployments import router as deployments_router
from .routes.health import router as health_router

router = APIRouter()
router.include_router(deployments_router)
router.include_router(health_router)

__all__ = ["router"]