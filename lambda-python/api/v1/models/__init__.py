"""
Package contenant les modèles Pydantic pour l'API v1.
"""
from .requests import DeploymentRequest, UpdateDeploymentRequest

__all__ = ["DeploymentRequest", "UpdateDeploymentRequest"]