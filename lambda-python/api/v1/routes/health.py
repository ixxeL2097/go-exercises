from fastapi import APIRouter, Depends
from services.kubernetes import KubernetesService

router = APIRouter(tags=["health"])

@router.get("/healthz")
async def health_check():
  return {"status": "healthy"}

@router.get("/readyz")
async def ready_check(
  k8s_service: KubernetesService = Depends()
):
  version = await k8s_service.get_api_version()
  return {
    "status": "ready",
    "kube API version": version.git_version
  }
