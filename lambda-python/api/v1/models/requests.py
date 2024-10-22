from pydantic import BaseModel, Field
from typing import Optional, Dict, Any

class DeploymentRequest(BaseModel):
  name: Optional[str] = Field(default=None, description="Name of the deployment")
  namespace: str = Field(..., description="Namespace of the deployment")
    
class UpdateDeploymentRequest(DeploymentRequest):
  annotations: Optional[Dict[str, str]] = Field(
    default=None,
    description="Annotations to update"
  )
  labels: Optional[Dict[str, str]] = Field(
    default=None,
    description="Labels to update"
  )