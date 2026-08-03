# PastureStack Node Agent

Node Agent receives control-platform events on compute nodes, executes the corresponding container, storage, host, and configuration operations, and publishes compatible replies.

PastureStack is an independent community effort to preserve, audit, and modernize the Rancher 1.6 ecosystem. It is not affiliated with or endorsed by Rancher Labs or SUSE.

**Upstream:** [`rancher/agent`](https://github.com/rancher/agent). This GitHub fork preserves upstream history, authorship, dates, tags, licenses, and bundled dependency notices; PastureStack maintenance is consolidated into one commit after the preserved upstream boundary.

## Project status

This is a migration proof of concept. The existing Ubuntu 26.04, Go 1.26.5, modern Docker test harness, runtime hardening, and dependency maintenance are retained. Product-owned import paths, binaries, archives, images, Windows service names, and operator messages use PastureStack naming. Python test dependencies are fully pinned and cached in the disposable build image so clean-checkout tests do not depend on live PyPI availability. Release packaging is manual; no CI/CD or automatic production deployment is enabled.

## Configuration

Preferred settings are `PLATFORM_URL`, `PLATFORM_ACCESS_KEY`, `PLATFORM_SECRET_KEY`, `PASTURESTACK_HOME`, and `PASTURESTACK_LOCALE`. The locale accepts `en-US` and `zh-TW`. Historical `CATTLE_*` settings are temporary compatibility aliases for established event and bootstrap contracts.

## Build and test

From a Docker-capable Linux host:

```sh
make test
make build
make package
make package-image IMAGE_NAME=pasturestack/node-agent TAG=poc
```

Packaging is local only and does not push an image. See [COMPATIBILITY.md](COMPATIBILITY.md), [SECURITY.md](SECURITY.md), and [ORIGIN.md](ORIGIN.md).

For the reviewed `0.13.22` compatibility release, `VERSION_OVERRIDE=v0.13.22 CROSS=1 make package` produces the deterministic flat assets `node-agent-0.13.22.tar.gz` and `node-agent-0.13.22-windows-amd64.zip`. PastureStack Server serves both from its matching GitHub Release and verifies their SHA-256 entries before use; operators do not need an artifact mirror. The Windows ZIP uses the neutral `pasturestack/` include layout. A replacement Windows bootstrap image and upgrade/rollback tests are still required before Windows hosts are supported.

The `host.port.check` event performs a read-only host-port preflight through the existing agent event channel. It reports Docker bindings from running and stopped containers and, on Linux, listening TCP/UDP sockets visible through the existing host `/proc` mount. Incomplete host socket inspection is reported as unknown; it is never presented as an available port.

The test harness starts a disposable inner Docker daemon and generates its BusyBox image, build contexts, and Git fixtures locally. It does not mount the host Docker socket or depend on mutable registry images and retired external test endpoints.

## License and attribution

The inherited project remains licensed under [Apache License 2.0](LICENSE). Copyright and attribution for inherited work and vendored dependencies remain with their respective authors and contributors. PastureStack contributors claim authorship only for their own changes.
