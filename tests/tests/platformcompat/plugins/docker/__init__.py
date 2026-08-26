import logging

from docker import APIClient as Client
from docker.utils import kwargs_from_env
from tests.platformcompat import default_value, Config

log = logging.getLogger('docker')

_ENABLED = True


def _compare_version(required, current):
    def parts(version):
        return [int(part) for part in version.split('.') if part.isdigit()]

    left = parts(current)
    right = parts(required)
    max_len = max(len(left), len(right))
    left += [0] * (max_len - len(left))
    right += [0] * (max_len - len(right))
    if left == right:
        return 0
    return 1 if left > right else -1


class DockerConfig:
    def __init__(self):
        pass

    @staticmethod
    def docker_enabled():
        return default_value('DOCKER_ENABLED', 'true') == 'true'

    @staticmethod
    def docker_host_ip():
        return default_value('DOCKER_HOST_IP', Config.agent_ip())

    @staticmethod
    def docker_home():
        return default_value('DOCKER_HOME', '/var/lib/docker')

    @staticmethod
    def docker_uuid_file():
        def_value = '{0}/.docker_uuid'.format(Config.state_dir())
        return default_value('DOCKER_UUID_FILE', def_value)

    @staticmethod
    def docker_uuid():
        return Config.get_uuid_from_file('DOCKER_UUID',
                                         DockerConfig.docker_uuid_file())

    @staticmethod
    def url_base():
        return default_value('DOCKER_URL_BASE', None)

    @staticmethod
    def api_version():
        return default_value('DOCKER_API_VERSION', '')

    @staticmethod
    def storage_api_version():
        return default_value('DOCKER_STORAGE_API_VERSION', '1.21')

    @staticmethod
    def docker_required():
        return default_value('DOCKER_REQUIRED', 'true') == 'true'

    @staticmethod
    def delegate_timeout():
        return int(default_value('DOCKER_DELEGATE_TIMEOUT', '120'))

    @staticmethod
    def use_boot2docker_connection_env_vars():
        use_b2d = default_value('DOCKER_USE_BOOT2DOCKER', 'false')
        return use_b2d.lower() == 'true'

    @staticmethod
    def is_host_pidns():
        return default_value('AGENT_PIDNS', 'container') == 'host'


def docker_client(version=None, base_url_override=None, tls_config=None,
                  timeout=None):
    if DockerConfig.use_boot2docker_connection_env_vars():
        try:
            kwargs = kwargs_from_env(assert_hostname=False)
        except TypeError:
            kwargs = kwargs_from_env()
    else:
        kwargs = {'base_url': DockerConfig.url_base()}

    if base_url_override:
        kwargs['base_url'] = base_url_override

    if tls_config:
        kwargs['tls'] = tls_config

    if version is None:
        version = DockerConfig.api_version()
    if version == '':
        version = 'auto'
    elif _compare_version('1.40', version) < 0:
        version = 'auto'

    if timeout:
        kwargs['timeout'] = timeout
    kwargs['version'] = version
    log.debug('docker client options configured')
    return Client(**kwargs)


try:
    Client
except NameError:
    log.info('Disabling docker, Docker SDK not found')
    _ENABLED = False
