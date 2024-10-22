from fastapi import HTTPException
from typing import Any

class KubernetesError(HTTPException):
  def __init__(self, detail: Any = None):
    super().__init__(status_code=500, detail=f"Kubernetes operation failed: {detail}")

class ResourceNotFoundError(HTTPException):
  def __init__(self, resource: str, name: str):
    super().__init__(
      status_code=404,
      detail=f"{resource} '{name}' not found"
    )