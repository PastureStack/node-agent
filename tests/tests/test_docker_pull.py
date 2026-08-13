from .common import docker_client, event_test, delete_container
import os
import pytest
from docker.errors import APIError


def ensure_busybox(client):
    try:
        client.inspect_image('busybox:1')
        return
    except APIError:
        pass

    image = os.environ.get('TEST_BUSYBOX_IMAGE', 'busybox:1')
    client.pull(image)
    if image != 'busybox:1':
        client.tag(image, 'busybox', '1', force=True)


def test_pull(agent):
    client = docker_client()
    try:
        client.remove_image('busybox:1random', force=True)
    except:
        pass

    def post(req, resp, valid_resp):
        inspect = client.inspect_image('busybox:1')
        assert resp['data']['fields']['dockerImage']['Id'] == inspect['Id']
        resp['data']['fields']['dockerImage'] = {}
        del resp['links']
        del resp['actions']
        del valid_resp['resourceId']

    event_test(agent, 'docker/instance_pull', post_func=post, diff=True)

    inspect = client.inspect_image('busybox:1random')
    assert inspect is not None

    def pre2(req):
        req['data']['instancePull']['complete'] = True

    def post2(req, resp, valid_resp):
        assert resp['data']['fields']['dockerImage']['Id'] == ''
        del resp['links']
        del resp['actions']
        del valid_resp['resourceId']

    event_test(agent, 'docker/instance_pull', pre_func=pre2, post_func=post2,
               diff=False)

    with pytest.raises(APIError):
        client.inspect_image('busybox:1random')


def test_pull_mode_update(agent):
    client = docker_client()

    with pytest.raises(APIError):
        client.inspect_image('garbage')

    def pre(req):
        req.data.instancePull.image.data.dockerImage.fullName = 'garbage'
        req.data.instancePull.mode = 'cached'

    def post(req, resp, valid_resp):
        assert resp['data']['fields']['dockerImage']['Id'] == ''
        del resp['links']
        del resp['actions']

    event_test(agent, 'docker/instance_pull', pre_func=pre, post_func=post,
               diff=False)


def test_pull_on_create(agent):
    client = docker_client()

    event_test(agent, 'docker/instance_activate', diff=False)
    delete_container('/c861f990-4472-4fa1-960f-65171b544c28')

    ensure_busybox(client)
    client.tag('busybox:1', 'ibuildthecloud/helloworld', 'latest',
               force=True)
    old_image = client.inspect_image('ibuildthecloud/helloworld')

    event_test(agent, 'docker/instance_activate_pull', diff=False)
    image = client.inspect_image('ibuildthecloud/helloworld')

    assert image['Id'] == old_image['Id']
    assert image['Id'] == client.inspect_image('busybox:1')['Id']
