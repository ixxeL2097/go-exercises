from fastapi import APIRouter, Depends
from core.logger import logger
from services.kubernetes import KubernetesService
from api.v1.models.requests import DeploymentRequest, UpdateDeploymentRequest

router = APIRouter(prefix="/deployments", tags=["deployments"])

@router.post("/restart")
async def restart_deployment(
  request: DeploymentRequest,
  k8s_service: KubernetesService =  Depends()
):
  """Restart a deployment by updating its pod template annotations."""
  # k8s_service = KubernetesService()
  logger.info(f"[ PROCESSING ] > Restarting deployment: {request.name} in {request.namespace}")
  
  await k8s_service.deployment_service.restart_deployment(request.name, request.namespace)
  
  return {
    "status": "success",
    "message": f"Deployment {request.name} in namespace {request.namespace} restarted successfully"
  }

@router.post("/list")
async def get_deployments(
  request: DeploymentRequest,
  k8s_service: KubernetesService =  Depends()
):
  """Getting list of deployments in a specific namespace."""
  # k8s_service = KubernetesService()
  logger.info(f"[ PROCESSING ] > Getting deployments in {request.namespace}")
  
  deployments = await k8s_service.deployment_service.list_deployments(request.namespace)
  deployments_name = [{"name": dep.metadata.name} for dep in deployments.items]
  return {
    "status": "success",
    "deployments": deployments_name
  }
