from kubernetes import client, config, dynamic
from core.logger import logger
from core.config import get_settings
from utils.exceptions import KubernetesError, ResourceNotFoundError
from typing import Tuple, Optional
import os
from datetime import datetime

settings = get_settings()

class KubernetesService:
  core_v1: client.CoreV1Api
  apps_v1: client.AppsV1Api
  version_api: client.VersionApi
  dynamic_client: dynamic.DynamicClient
  deployment_service: 'DeploymentService'

  def __init__(self):
    self._init_client()
    self.deployment_service = self.DeploymentService(self)
        
  def _init_client(self):
    try:
      if settings.IN_CLUSTER:
        config.load_incluster_config()
      else:
        config.load_kube_config(settings.KUBE_CONFIG_PATH)
          
      self.core_v1 = client.CoreV1Api()
      self.apps_v1 = client.AppsV1Api()
      self.version_api = client.VersionApi()
      self.dynamic_client = dynamic.DynamicClient(
          client.api_client.ApiClient()
      )
    except Exception as e:
      logger.error(f"Failed to initialize Kubernetes client: {e}")
      raise KubernetesError(str(e))
    
  async def get_api_version(self) -> client.VersionInfo:
    try:
      version = self.version_api.get_code()
      return version
    except Exception as e:
      logger.error(f"Cannot get API version")
      raise KubernetesError(str(e))
    
  class DeploymentService:
    def __init__(self, kubernetes_service: 'KubernetesService'):
      self.k8s_service = kubernetes_service

    async def list_deployments(self, namespace: str) -> client.V1DeploymentList:
      try:
        deployments = self.k8s_service.apps_v1.list_namespaced_deployment(namespace=namespace, pretty='true')
        return deployments
      except Exception as e:
        logger.error(f"Failed to get deployments in namespace {namespace}: {e}")
        raise KubernetesError(str(e))

    async def get_deployment(self, name: str, namespace: str) -> client.V1Deployment:
      try:
        return await self.k8s_service.apps_v1.read_namespaced_deployment(name, namespace)
      except client.exceptions.ApiException as e:
        if e.status == 404:
          raise ResourceNotFoundError("Deployment", name)
        raise KubernetesError(str(e))

    async def restart_deployment(self, name: str, namespace: str):
      try:
        deployment = await self.get_deployment(name, namespace)

        if deployment.spec.template.metadata is None:
          deployment.spec.template.metadata = client.V1ObjectMeta()

        annotations = deployment.spec.template.metadata.annotations or {}
        annotations["kubectl.kubernetes.io/restartedAt"] = datetime.utcnow().strftime('%Y-%m-%dT%H:%M:%SZ')

        deployment.spec.template.metadata.annotations = annotations

        return await self.k8s_service.apps_v1.patch_namespaced_deployment(
          name=name,
          namespace=namespace,
          body=deployment
        )
      except Exception as e:
        logger.error(f"Failed to restart deployment {name}: {e}")
        raise KubernetesError(str(e))