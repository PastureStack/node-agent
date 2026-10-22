# Compatibility Contract

The node agent preserves established event names, reply payloads, generated client schemas, Docker labels, state fields, registration-script inputs, and storage-driver identifiers consumed by existing deployments.

## Linux Host API producer in 0.13.31

The source package version is `0.13.31`. Its Linux supervisor requires exactly
Host API `0.38.7`, installed first through the existing authenticated configcontent
package sequence. Version mismatch still closes Host API admission without a
legacy fallback. Fixed producer paths, agent identity forwarding, executable
ownership checks, parent-bound lifetime, package layout, and both SHA-1 and
SHA-256 verification contracts are unchanged. Node does not implement stream
authorization or terminal-cause arbitration.

Linux and Windows cross-packaging share this source version. Windows retains its
existing embedded compatibility path; cross-builds are not Windows runtime or
delegated-stream acceptance. Source versions are not publication evidence.

## Linux Host disk reporting in 0.13.29

Filesystem enumeration and usage measurement use the same process mount view.
Namespace detection reads only a fixed namespace identifier, without following
its magic-link target. Direct-host execution, custom proc roots and existing
mount-table fallback remain supported. Unreadable mounts preserve other valid
data and report a warning; read-only Snap images do not inflate filesystem totals.
`localStorageMb` is available space in whole MiB, not total capacity; explicit
`CATTLE_LOCAL_STORAGE_MB_OVERRIDE` keeps precedence. Schemas, scheduler behavior,
Host API supervision and event authorization are unchanged.

The [immutable Linux release](https://github.com/PastureStack/node-agent/releases/tag/v0.13.29)
uses producer `514a3c1605809628693430007b30a33f4b1efbb4` and the original
[CI 39411416316](https://github.com/PastureStack/node-agent/actions/runs/39411416316)
package. Focused tests cover namespace views, partial errors, proc fallback,
magic links, malformed identifiers, parent-link escape and MiB boundaries.
Required CodeQL passed without dismissing alerts or bypassing checks. Windows
cross-packaging is not new Windows runtime certification; the independently
pinned Server Windows artifact remains unchanged.

Preferred PastureStack settings use `PLATFORM_*` and `PASTURESTACK_HOME`. Historical `CATTLE_*`, `io.rancher.*`, internal DNS names, generated `RancherClient` types, and vendored `github.com/rancher/*` paths remain only where they are protocol, data, or inherited dependency contracts. They are not PastureStack branding and must not be mechanically removed.

Operator lifecycle messages support `PASTURESTACK_LOCALE=en-US` and `PASTURESTACK_LOCALE=zh-TW`. Event payloads, resource identifiers, labels, errors returned through the API, and Docker output are deliberately not translated.

Before release, validate registration, restart, container lifecycle, CNI metadata, DNS search, storage attach/mount/unmount, Windows service migration, and rollback against an isolated compatibility control plane.

Current Moby no longer accepts the legacy standalone `kernelMemory` and
`diskQuota` create fields. Node Agent keeps accepting those historical event
fields as logged no-ops so an old control plane cannot block otherwise valid
container creation. PID limits, DNS addresses, port bindings, and MAC addresses
are translated to the current Moby resource, `netip`, port-map, and endpoint
types.
