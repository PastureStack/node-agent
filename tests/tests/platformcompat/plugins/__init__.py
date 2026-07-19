import importlib
import logging
import os


log = logging.getLogger("agent")


def _load(module):
    full_name = "tests.platformcompat.plugins.%s" % module
    log.info("Loading Plugin: %s", full_name)
    try:
        return importlib.import_module(full_name)
    except Exception:
        log.exception('Exception loading module')


def _init(full_path):
    for name in os.listdir(full_path):
        plugin_path = os.path.join(full_path, name)
        if os.path.exists(os.path.join(plugin_path, "__init__.py")):
            _load(name)


def load():
    _init(os.path.dirname(os.path.abspath(__file__)))
