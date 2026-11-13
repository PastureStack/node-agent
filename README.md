# PastureStack Node Agent

The published Linux runtime **`v0.13.37`** requires Host API **`0.38.10`**. The Linux and Windows packages and OCI image were built with Go 1.27.4 and verified by the [release pipeline](https://github.com/PastureStack/node-agent/actions/runs/43355589004).

The runtime integrates shared Ubuntu 26.04 security backports for already-installed packages while preserving the selected coreutils provider.

The image retains bundle manifests, source inputs and installation receipts under `/usr/share/pasturestack/security/ubuntu-backports/`.

Node Agent receives control-platform events on compute nodes, executes the corresponding container, storage, host, and configuration operations, and publishes compatible replies.

PastureStack is an independent community effort to preserve, audit, and modernize the Rancher 1.6 ecosystem. It is not affiliated with or endorsed by Rancher Labs or SUSE.

**Upstream:** [`rancher/agent`](https://github.com/rancher/agent). This GitHub fork preserves upstream history, authorship, dates, tags, licenses, and bundled dependency notices.

## Project status

This is a migration proof of concept. The existing Ubuntu 26.04, modern Docker test harness, runtime hardening, and dependency maintenance are retained, with the compiler updated to checksum-pinned Go 1.27.4. Product-owned import paths, binaries, archives, images, Windows service names, and operator messages use PastureStack naming. Python test dependencies are fully pinned and cached in the disposable build image so clean-checkout tests do not depend on live PyPI availability. GitHub Actions runs repository validation and security checks. Release packaging remains manual, and no automatic production deployment is enabled.

## Configuration

Preferred settings are `PLATFORM_URL`, `PLATFORM_ACCESS_KEY`, `PLATFORM_SECRET_KEY`, `PASTURESTACK_HOME`, and `PASTURESTACK_LOCALE`. The locale accepts `en-US` and `zh-TW`. Historical `CATTLE_*` settings are temporary compatibility aliases for established event and bootstrap contracts.

## Build and test

From a Docker-capable Linux host:

```sh
make test
make build
make package
make package-image IMAGE_NAME=pasturestack/node-agent TAG=v0.13.37
```

Packaging is local only and does not push an image. See [COMPATIBILITY.md](COMPATIBILITY.md), [SECURITY.md](SECURITY.md), and [ORIGIN.md](ORIGIN.md).

The recipe packages version `0.13.37`. Verify the [published assets](https://github.com/PastureStack/node-agent/releases/tag/v0.13.37)
using their SHA-256 checksums. Published tags and assets
remain immutable; later documentation commits do not change their producer identity.
The outer bootstrap image version and the Go package version are separate identities.

The authoritative build is Go modules with the checked-in `vendor/` tree. The
runtime dependency baseline uses Go 1.27, AWS SDK for Go v2, Moby API 1.55 and
client 0.5, gopsutil v4.26, mapstructure v2.5, netlink v1.3, and netns v0.0.5.
The retired AWS SDK v1, Aliyungo, root Docker module, GOPATH/Godeps, and Trash
dependency paths are not part of the build.

`VERSION_OVERRIDE=v0.13.37 make package` produces the Linux asset `node-agent-0.13.37.tar.gz`. The archive carries the legacy SHA-1 and current SHA-256 manifests required by the authenticated configcontent installer. The producer release must pass this repository's normal publication and byte verification before Server can embed that archive and distribute it through the existing configuration channel. Component publication and matching Server acceptance are separate evidence boundaries.

Upgrade existing Linux hosts through that established configcontent/pyagent
sequence, installing the compatible Host API package before Node. Keep the
existing platform-home data volume and Host ID; do not register a replacement
host or create a replacement state volume. Read the installed Node version with
`${PASTURESTACK_HOME:-${CATTLE_HOME:-/var/lib/pasturestack}}/node-agent/node-agent --version`;
after installing this Linux version, the expected result is
`node-agent version v0.13.37`.

On Linux, Node supervises the separately maintained `host-api` producer at `${PASTURESTACK_HOME:-${CATTLE_HOME:-/var/lib/pasturestack}}/bin/host-api`. Host API `0.38.10` must be installed before the Node package by the existing `pyagent` package sequence. Node verifies its exact version, forwards the current agent/host identity only through process environment, restarts an exited child, and binds its lifetime to Node. Missing or incompatible producers never fall back to the embedded legacy Host API. Stream authorization, target binding, and private durable terminal evidence remain owned by the Host API producer.

Windows retains the existing embedded compatibility path; this Linux image preparation does not rebuild or revalidate the separately pinned Windows runtime or provide Windows stream delegation or durable terminal evidence support. A replacement Windows bootstrap image and upgrade/rollback tests are still required before Windows hosts are supported. The Server recipe independently pins its Windows artifact; a Linux package publication and CI cross-builds do not establish new Windows runtime acceptance.

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

Maintained Go builds select the patched Go 1.27.4 compiler without raising the existing Go language requirement. Ubuntu build/runtime stages apply the checksum-pinned `ca-certificates 20260816ubuntu0.26.04.2` security package after their frozen APT baseline; the final trust store and installed version are verified. Python tooling uses the patched pip/CA path; CI interpreter pins remain on the Python 3.14 series. Release artifacts must pass the repository's existing checks before publication.
# Python installer security rebuild

The disposable Dapper image rebuilds pip `26.2.2+pasturestack.2` from checksum-pinned official inputs using upstream vendoring and the original urllib3 patches. urllib3 uses released `2.8.0` and msgpack uses released `1.2.3`; this is a local security rebuild, not a new official PyPA release. The expected wheel SHA-256 is `2aef7c341eef310645ab56455f5789de2819463b25fa70151954e1d208d2db36`. Ubuntu's Python ABI is retained. The image tests live pip, available ensurepip/new-venv seeds, virtualenv, streaming limits, and trusted/untrusted TLS. See `scripts/pip-security/README.md` for source and license evidence.

## Full-severity security rebuild

The recipe updates only already-installed Ubuntu 26.04 packages and obtains
the trust bundle from the SHA-pinned official CA package. Host ID, persistent
agent state and startup arguments remain unchanged. This release advances the
exact required Host API producer from 0.38.8 to 0.38.9.
Use the selected release's checksum-covered assets and provenance to verify
its producer and immutable image bytes.

The full-severity runtime gate rejects every finding with an official fix,
including LOW, MEDIUM and UNKNOWN. Only exact reviewed LOW/MEDIUM no-fix entries
may pass. The original all-severity reports, exact no-fix decisions and
binary-specific package-graph evidence are retained independently.
### Complete build-environment gate

The final runtime installs checksum-pinned official APT `3.2.1ubuntu0.1`.
The disposable builder installs official PCRE2 `10.46-1ubuntu0.1`; the runtime
retains its independently qualified shared PCRE2 backport. CI compares both the
complete pip core and vendor file sets with the selected patched wheel.

The builder installs checksum-pinned official Ubuntu `linux-libc-dev 7.0.0-43.43`.
The exact existing nonexecuted-header claims remain separate from the 302
Medium findings recorded as `tracked_unfixed` in
`.github/dapper-header-tracked-unfixed.json`. Those findings remain in the raw
scan and are not declared code-absent. The record binds the official package and
original header-only scan, with a UTC next-review date of 2026-11-14. An available
fix, changed severity/identity or unregistered finding still blocks release.


The Dapper scan retains all severities and requires nonempty package analysis for the expected Go binaries. Any available fixed version fails, including Low, Medium and Unknown findings. Other Low/Medium findings require an exact package/version/PURL entry backed by the official vendor tracker. The pinned userspace kernel-header exception additionally requires the reviewed CVE set, actual header payload paths, and absence of kernel images/modules; it never permits an available fix. Unknown module findings require the separately verified exact source package graph, not a generic severity exemption.

The disposable builder uses official Docker 29.9.3 CLI/daemon bytes, with its
supported Go 1.26.10 compiler, rather than Ubuntu's outdated embedded Go tools.
Containerd stays on the 2.2 compatibility branch at 2.2.10 and runc stays on 1.4
at 1.4.3; both are rebuilt from SHA-pinned released sources using Go 1.27.4 and
released security-fixed dependencies. Runc retains seccomp. Containerd
uses released imgcrypt 2.0.3 / ocicrypt 1.3.2 to retain encrypted-image PGP
support through the maintained ProtonMail implementation; CI tests that wrapper
and requires the deprecated x/crypto OpenPGP package to be absent.
It omits the unused Btrfs snapshotter; the actual test daemon uses overlay2/vfs.
These are builder-only changes, not a host Docker upgrade or a new deployment
recipe. CI exercises the real inner daemon and package-image build, independently
scans all seven container-tool ELF files, and fails if any expected analysis is
missing. Complete raw image and standalone ELF reports are retained.

The daemon's reachable distribution metadata API retains CVE-2026-85747 as an
exact reviewed Medium no-fix finding: the official advisory has no released
docker/distribution v2 fix. The v3.1.2 fix is a different module major; neither an
unpublished v2 commit nor the CLI's digest mitigation is substituted. Future
official fixed-version findings remain blocking. The exact official daemon's
`golang.org/x/crypto/openpgp` absence is separately checked against its SHA and
nonempty crypto symbol control; it does not exempt the distribution finding.
The retained 2,080-package daemon source graph is from official commit
`70efc0e31da3a115fc87574d1b786fa2b867c346`, Linux/amd64, CGO=1 and official tags
`netgo,osusergo,static_build,nri_no_wasm,journald`, resolved offline from the
official vendor tree using Go 1.27.0 for this static source review. That analysis compiler is not the official
binary's Go 1.26.10 compiler. The graph includes ProtonMail's distinct OpenPGP
implementation; no claim is made that all OpenPGP code is absent.
