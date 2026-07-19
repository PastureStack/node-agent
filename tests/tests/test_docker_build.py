from functools import partial
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer
import io
import subprocess
import tarfile
import threading

import pytest

from tests.common import delete_container, JsonObject, \
    instance_activate_common_validation, event_test
from tests.platformcompat.utils import random_string


@pytest.fixture(scope='module')
def build_context_urls(tmp_path_factory):
    root = tmp_path_factory.mktemp('build-contexts')
    source = root / 'source'
    source.mkdir()
    dockerfile = (
        b'FROM busybox:1\n'
        b'RUN echo pasturestack-fixture > /fixture.txt\n'
    )
    (source / 'Dockerfile').write_bytes(dockerfile)

    subprocess.check_call(['git', 'init', str(source)])
    subprocess.check_call(['git', '-C', str(source), 'config',
                           'user.name', 'PastureStack Test'])
    subprocess.check_call(['git', '-C', str(source), 'config',
                           'user.email', 'test@example.invalid'])
    subprocess.check_call(['git', '-C', str(source), 'add', 'Dockerfile'])
    subprocess.check_call(['git', '-C', str(source), 'commit', '-m',
                           'Add deterministic build fixture'])
    bare = root / 'tiny-build.git'
    subprocess.check_call(['git', 'clone', '--bare', str(source), str(bare)])
    subprocess.check_call(['git', '-C', str(bare), 'update-server-info'])

    tar_path = root / 'build.tar'
    info = tarfile.TarInfo('Dockerfile')
    info.size = len(dockerfile)
    info.mode = 0o644
    info.mtime = 0
    with tarfile.open(str(tar_path), 'w') as archive:
        archive.addfile(info, io.BytesIO(dockerfile))

    handler = partial(SimpleHTTPRequestHandler, directory=str(root))
    server = ThreadingHTTPServer(('127.0.0.1', 0), handler)
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        base = 'http://127.0.0.1:%d' % server.server_port
        yield base + '/tiny-build.git', base + '/build.tar'
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


def _test_docker_build_from_remote(agent, remote=None,
                                   context=None):

    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    image_uuid = 'image-' + random_string(12)

    def pre(req):
        instance = req.data.instanceHostMap.instance
        # tag is not on the instance, only the image
        instance.data.fields['build'] = JsonObject({
            'remote': remote,
            'context': context,
        })
        instance.data.fields.imageUuid = image_uuid
        instance.image.data['fields'] = JsonObject({
            'build': {
                'remote': remote,
                'context': context,
                'tag': image_uuid,
            },
        })

    def post(req, resp):
        instance_data = resp['data']['instanceHostMap']['instance']['+data']
        docker_inspect = instance_data['dockerInspect']
        image = docker_inspect['Config']['Image']
        assert image_uuid == image
        instance_activate_common_validation(resp)

    event_test(agent, 'docker/instance_activate', diff=False,
               pre_func=pre, post_func=post)

    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')


def test_docker_build_from_git(agent, build_context_urls):
    remote, _ = build_context_urls
    _test_docker_build_from_remote(agent, remote)


def test_docker_build_from_context(agent, build_context_urls):
    _, url = build_context_urls
    _test_docker_build_from_remote(agent, context=url)
