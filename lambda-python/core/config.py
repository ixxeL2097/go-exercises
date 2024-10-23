from pydantic_settings import BaseSettings
from functools import lru_cache

class Settings(BaseSettings):
  APP_NAME: str = "k8s-service"
  APP_VERSION: str = "1.0.0"
  APP_PORT: int = 8081
  LOG_LEVEL: str = "INFO"
  
  # K8s settings
  KUBE_CONFIG_DEFAULT_LOCATION: str = "~/.kube/config"
  KUBE_CONFIG_PATH: str | None = None
  IN_CLUSTER: bool = False
  
  class Config:
    env_file = ".env"

@lru_cache()
def get_settings() -> Settings:
  return Settings()