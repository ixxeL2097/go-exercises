from core.config import get_settings
from kubernetes.config import load_kube_config, load_incluster_config
from os.path import exists, expanduser
from core.logger import logger

settings = get_settings()

def load_config(**kwargs):
  """
  Wrapper function to load the kube_config.
  It will initially try to load_kube_config from provided path,
  then check if the KUBE_CONFIG_DEFAULT_LOCATION exists
  If neither exists, it will fall back to load_incluster_config
  and inform the user accordingly.

  :param kwargs: A combination of all possible kwargs that
  can be passed to either load_kube_config or
  load_incluster_config functions.
  """
  if "config_file" in kwargs.keys():
    logger.info(f"[ INITIALIZATION ] > Loading kubeconfig from config_file file {kwargs.get('config_file')}")
    load_kube_config(**kwargs)
  elif "kube_config_path" in kwargs.keys():
    kwargs["config_file"] = kwargs.pop("kube_config_path", None)
    logger.info(f"[ INITIALIZATION ] > Loading kubeconfig from kube_config_path file {kwargs.get('kube_config_path')}")
    load_kube_config(**kwargs)
  elif exists(expanduser(settings.KUBE_CONFIG_DEFAULT_LOCATION)):
    logger.info(f"[ INITIALIZATION ] > Loading kubeconfig from file {settings.KUBE_CONFIG_DEFAULT_LOCATION}")
    load_kube_config(**kwargs)
  else:
    print(
      "kube_config_path not provided and "
      "default location ({0}) does not exist. "
      "Using inCluster Config. "
      "This might not work.".format(settings.KUBE_CONFIG_DEFAULT_LOCATION))
    logger.info(f"[ INITIALIZATION ] > Loading in cluster config")
    load_incluster_config(**kwargs)