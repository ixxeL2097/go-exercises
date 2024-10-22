from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from fastapi.middleware.cors import CORSMiddleware
from core.config import get_settings
from core.logger import logger
from api.v1.routes import deployments, health
from utils.exceptions import KubernetesError, ResourceNotFoundError
import uvicorn

settings = get_settings()

def create_app() -> FastAPI:
  app = FastAPI(
    title=settings.APP_NAME,
    version=settings.APP_VERSION,
    description="Kubernetes Management API"
  )
  
  # CORS middleware
  app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
  )
  
  # Exception handlers
  @app.exception_handler(KubernetesError)
  async def kubernetes_error_handler(request: Request, exc: KubernetesError):
    return JSONResponse(
      status_code=exc.status_code,
      content={"detail": exc.detail}
    )
  
  @app.exception_handler(ResourceNotFoundError)
  async def not_found_error_handler(request: Request, exc: ResourceNotFoundError):
    return JSONResponse(
      status_code=exc.status_code,
      content={"detail": exc.detail}
    )
  
  # Include routers
  app.include_router(health.router)
  app.include_router(deployments.router, prefix="/v1")
  
  return app

app = create_app()

if __name__ == "__main__":
  uvicorn.run(
    "main:app",
    host="0.0.0.0",
    port=settings.APP_PORT,
    reload=True
  )