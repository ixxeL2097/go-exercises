import logging
import sys
from core.config import get_settings

settings = get_settings()

def setup_logger():
  logger = logging.getLogger("uvicorn.error")
  logger.setLevel(settings.LOG_LEVEL)
  
  formatter = logging.Formatter(
    '%(asctime)s - %(name)s - %(levelname)s - %(message)s'
  )
  
  handler = logging.StreamHandler(sys.stdout)
  handler.setFormatter(formatter)
  logger.addHandler(handler)
  
  return logger

logger = setup_logger()