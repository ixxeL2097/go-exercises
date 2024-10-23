import threading

class Singleton:
  _instance = None
  _lock = threading.Lock()

  def __new__(cls, *args, **kwargs):
    if not cls._instance:
      with cls._lock:
        if not cls._instance:
          cls._instance = super(Singleton, cls).__new__(cls)
          cls._instance.__initialized = False
    return cls._instance
