from urllib.parse import urlparse
import logging
import socket

from tests.platformcompat import Config
from tests.platformcompat.utils import get_url_port
from tests.platformcompat.process_manager import background


log = logging.getLogger('api-proxy')


class ApiProxy(object):
    def __init__(self):
        self.pid = None

    def on_startup(self):
        url = Config.config_url()

        if 'localhost' not in url:
            return

        parsed = urlparse(url)

        from_host = Config.api_proxy_listen_host()
        from_port = Config.api_proxy_listen_port()
        to_host_ip = socket.gethostbyname(parsed.hostname)
        to_port = get_url_port(url)

        log.info('Starting local API proxy')
        listen = 'TCP4-LISTEN:{0},fork,bind={1},reuseaddr'.format(from_port,
                                                                  from_host)
        to = 'TCP:{0}:{1}'.format(to_host_ip, to_port)

        background(['socat', listen, to])
