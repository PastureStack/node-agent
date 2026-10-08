# PastureStack Node Agent

Node Agent receives control-platform events on compute nodes, executes the corresponding container, storage, host, and configuration operations, and publishes compatible replies.

PastureStack is an independent community effort to preserve, audit, and modernize the Rancher 1.6 ecosystem. It is not affiliated with or endorsed by Rancher Labs or SUSE.

**Upstream:** [`rancher/agent`](https://github.com/rancher/agent). This GitHub fork preserves upstream history, authorship, dates, tags, licenses, and bundled dependency notices; PastureStack maintenance is consolidated into one commit after the preserved upstream boundary.

## Project status

This is a migration proof of concept. The existing Ubuntu 26.04, Go 1.27.0, modern Docker test harness, runtime hardening, and dependency maintenance are retained. Product-owned import paths, binaries, archives, images, Windows service names, and operator messages use PastureStack naming. Python test dependencies are fully pinned and cached in the disposable build image so clean-checkout tests do not depend on live PyPI availability. GitHub Actions runs repository validation and security checks. Release packaging remains manual, and no automatic production deployment is enabled.

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

The authoritative build is Go modules with the checked-in `vendor/` tree. The
runtime dependency baseline uses Go 1.27, AWS SDK for Go v2, Moby API 1.55 and
client 0.5, gopsutil v4.26, mapstructure v2.5, netlink v1.3, and netns v0.0.5.
The retired AWS SDK v1, Aliyungo, root Docker module, GOPATH/Godeps, and Trash
dependency paths are not part of the build.

`VERSION_OVERRIDE=v0.13.28 make package` produces the deterministic Linux asset `node-agent-0.13.28.tar.gz`. The archive carries the legacy SHA-1 and current SHA-256 manifests required by the authenticated configcontent installer. The producer release is published by this repository; Server can embed that verified archive and distribute it through the existing configuration channel.

On Linux, Node supervises the separately maintained `host-api` producer at `${PASTURESTACK_HOME:-${CATTLE_HOME:-/var/lib/pasturestack}}/bin/host-api`. Host API `0.38.5` must be installed before the Node package by the existing `pyagent` package sequence. Node verifies its exact version, forwards the current agent/host identity only through process environment, restarts an exited child, and binds its lifetime to Node. Missing or incompatible producers never fall back to the embedded legacy Host API. Stream authorization, target binding, and private durable terminal evidence remain owned by the Host API producer.

Windows retains the existing embedded compatibility path; this release does not provide Windows stream delegation or durable terminal evidence support. A replacement Windows bootstrap image and upgrade/rollback tests are still required before Windows hosts are supported. Existing Windows release pins should remain unchanged.

The `host.port.check` event performs a read-only host-port preflight through the existing agent event channel. It reports Docker bindings from running and stopped containers and, on Linux, listening TCP/UDP sockets visible through the existing host `/proc` mount. Incomplete host socket inspection is reported as unknown; it is never presented as an available port.

The test harness starts a disposable inner Docker daemon and generates its BusyBox image, build contexts, and Git fixtures locally. It does not mount the host Docker socket or depend on mutable registry images and retired external test endpoints.

## License and attribution

The inherited project remains licensed under [Apache License 2.0](LICENSE). Copyright and attribution for inherited work and vendored dependencies remain with their respective authors and contributors. PastureStack contributors claim authorship only for their own changes.
