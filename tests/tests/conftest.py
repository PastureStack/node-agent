import pytest
import logging
import os
from os.path import dirname
import os.path
import requests
from subprocess import Popen, STDOUT
import time
from tests.common import TEST_DIR, Agent

logging.basicConfig()
log = logging.getLogger("tests")
log.setLevel(logging.INFO)

PROJECT_DIR = dirname(dirname(TEST_DIR))
GOPATH_DIR = dirname(dirname(dirname(dirname(PROJECT_DIR))))


@pytest.fixture(scope='session', autouse=True)
def start_server(request):
    env = dict(os.environ)
    env["GOPATH"] = GOPATH_DIR
    env["GO111MODULE"] = "off"
    log_path = os.path.join(TEST_DIR, "test-event-server.log")
    log_file = open(log_path, "w")
    proc = Popen(["go", "run", os.path.join(dirname(TEST_DIR), "main.go")],
                 cwd=dirname(TEST_DIR), env=env,
                 stdout=log_file, stderr=STDOUT)

    def kill_server():
        try:
            requests.get("http://localhost:8089/die")
        except Exception:
            pass
        if proc.poll() is None:
            proc.terminate()
            proc.wait()
        log_file.close()
    request.addfinalizer(kill_server)

    wait = .25
    max_wait = 2
    startup_timeout = float(
        os.environ.get("PASTURESTACK_TEST_EVENT_SERVER_TIMEOUT", "180"))
    deadline = time.monotonic() + startup_timeout
    while True:
        try:
            requests.get("http://localhost:8089/ping")
        except Exception:
            if proc.poll() is not None:
                log_file.flush()
                with open(log_path) as event_log:
                    pytest.fail("test event server exited early:\n%s" %
                                event_log.read())
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                log.error("Timed out waiting for test event server")
                log_file.flush()
                with open(log_path) as event_log:
                    pytest.fail(
                        "Timed out waiting for test event server:\n%s" %
                        event_log.read())
            sleep_for = min(wait, remaining)
            log.info("Waiting %ss on test event server (%ss remaining)" %
                     (sleep_for, max(0, remaining)))
            time.sleep(sleep_for)
            wait = min(wait * 2, max_wait)
        else:
            break


@pytest.fixture(scope="module")
def agent():
    return Agent()
