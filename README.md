# PastureStack Node Agent

Node Agent receives control-platform events on compute nodes, executes the corresponding container, storage, host, and configuration operations, and publishes compatible replies.

PastureStack is an independent community effort to preserve, audit, and modernize the Rancher 1.6 ecosystem. It is not affiliated with or endorsed by Rancher Labs or SUSE.

**Upstream:** [`rancher/agent`](https://github.com/rancher/agent). This GitHub fork preserves upstream history, authorship, dates, tags, licenses, and bundled dependency notices.

## Project status

This is a migration proof of concept. The existing Ubuntu 26.04, modern Docker test harness, runtime hardening, and dependency maintenance are retained, with the compiler updated to checksum-pinned Go 1.27.3. Product-owned import paths, binaries, archives, images, Windows service names, and operator messages use PastureStack naming. Python test dependencies are fully pinned and cached in the disposable build image so clean-checkout tests do not depend on live PyPI availability. GitHub Actions runs repository validation and security checks. Release packaging remains manual, and no automatic production deployment is enabled.

## Configuration

Preferred settings are `PLATFORM_URL`, `PLATFORM_ACCESS_KEY`, `PLATFORM_SECRET_KEY`, `PASTURESTACK_HOME`, and `PASTURESTACK_LOCALE`. The locale accepts `en-US` and `zh-TW`. Historical `CATTLE_*` settings are temporary compatibility aliases for established event and bootstrap contracts.

## Build and test

From a Docker-capable Linux host:

```sh
make test
make build
make package
make package-image IMAGE_NAME=pasturestack/node-agent TAG=v0.13.32
```

Packaging is local only and does not push an image. See [COMPATIBILITY.md](COMPATIBILITY.md), [SECURITY.md](SECURITY.md), and [ORIGIN.md](ORIGIN.md).

The current recipe packages version `0.13.32`. Source changes alone are not
publication proof: use the [official releases](https://github.com/PastureStack/node-agent/releases)
and their SHA-256 checksums to verify published packages. Published tags and assets
remain immutable; later documentation commits do not change their producer identity.
The outer bootstrap image version and the Go package version are separate identities.

The authoritative build is Go modules with the checked-in `vendor/` tree. The
runtime dependency baseline uses Go 1.27, AWS SDK for Go v2, Moby API 1.55 and
client 0.5, gopsutil v4.26, mapstructure v2.5, netlink v1.3, and netns v0.0.5.
The retired AWS SDK v1, Aliyungo, root Docker module, GOPATH/Godeps, and Trash
dependency paths are not part of the build.

`VERSION_OVERRIDE=v0.13.32 make package` produces the deterministic Linux asset `node-agent-0.13.32.tar.gz`. The archive carries the legacy SHA-1 and current SHA-256 manifests required by the authenticated configcontent installer. The producer release is published by this repository; Server can embed that verified archive and distribute it through the existing configuration channel. Component publication and matching Server acceptance are separate evidence boundaries.

On Linux, Node supervises the separately maintained `host-api` producer at `${PASTURESTACK_HOME:-${CATTLE_HOME:-/var/lib/pasturestack}}/bin/host-api`. Host API `0.38.7` must be installed before the Node package by the existing `pyagent` package sequence. Node verifies its exact version, forwards the current agent/host identity only through process environment, restarts an exited child, and binds its lifetime to Node. Missing or incompatible producers never fall back to the embedded legacy Host API. Stream authorization, target binding, and private durable terminal evidence remain owned by the Host API producer.

Windows retains the existing embedded compatibility path; this release does not provide Windows stream delegation or durable terminal evidence support. A replacement Windows bootstrap image and upgrade/rollback tests are still required before Windows hosts are supported. The Server recipe independently pins its Windows artifact; a Linux package publication and CI cross-builds do not establish new Windows runtime acceptance.

The `host.port.check` event performs a read-only host-port preflight through the existing agent event channel. It reports Docker bindings from running and stopped containers and, on Linux, listening TCP/UDP sockets visible through the existing host `/proc` mount. Incomplete host socket inspection is reported as unknown; it is never presented as an available port.

Linux disk information reads filesystem usage through the root of the process
whose mounts are enumerated, including when the agent has a separate mount
namespace. Direct host execution retains direct paths. Read-only Snap filesystem
images are excluded from disk capacity; an unreadable mount reports a collection
warning while preserving other successfully collected filesystems and mount points.
Host `localStorageMb` reports available space in whole MiB, not total disk capacity;
`CATTLE_LOCAL_STORAGE_MB_OVERRIDE` retains priority over the measured value.

The test harness starts a disposable inner Docker daemon and generates its BusyBox image, build contexts, and Git fixtures locally. It does not mount the host Docker socket or depend on mutable registry images and retired external test endpoints.

## License and attribution

The inherited project remains licensed under [Apache License 2.0](LICENSE). Copyright and attribution for inherited work and vendored dependencies remain with their respective authors and contributors. PastureStack contributors claim authorship only for their own changes.

## October 2026 security maintenance

Maintained Go builds select the patched Go 1.27.3 compiler without raising the existing Go language requirement. Ubuntu build/runtime stages apply the checksum-pinned `ca-certificates 20260816ubuntu0.26.04.1` security package after their frozen APT baseline; the final trust store and installed version are verified. Python tooling uses the patched pip/CA path; CI interpreter pins remain on the Python 3.14 series. Release artifacts must pass the repository's existing checks before publication.
# Python installer security rebuild

The disposable Dapper image rebuilds pip `26.2.2+pasturestack.1` from checksum-pinned official inputs using upstream vendoring and the original urllib3 patches. Only urllib3 changes to released `2.8.0`; this is a local security rebuild, not a new official PyPA release. The expected wheel SHA-256 is `be5f63d537bbe611dc422fabb6740a44b5997888b592888df5091727f023802a`. Ubuntu's Python ABI is retained. The image tests live pip, available ensurepip/new-venv seeds, virtualenv, streaming limits, and trusted/untrusted TLS. See `scripts/pip-security/README.md` for source and license evidence.
