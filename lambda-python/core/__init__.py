"""
Package core contenant la configuration et les utilitaires principaux.
"""
from .config import get_settings
from .logger import logger

__all__ = ["get_settings", "logger"]