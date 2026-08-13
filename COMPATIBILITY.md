# Compatibility Contract

The node agent preserves established event names, reply payloads, generated client schemas, Docker labels, state fields, registration-script inputs, and storage-driver identifiers consumed by existing deployments.

Preferred PastureStack settings use `PLATFORM_*` and `PASTURESTACK_HOME`. Historical `CATTLE_*`, `io.rancher.*`, internal DNS names, generated `RancherClient` types, and vendored `github.com/rancher/*` paths remain only where they are protocol, data, or inherited dependency contracts. They are not PastureStack branding and must not be mechanically removed.

Operator lifecycle messages support `PASTURESTACK_LOCALE=en-US` and `PASTURESTACK_LOCALE=zh-TW`. Event payloads, resource identifiers, labels, errors returned through the API, and Docker output are deliberately not translated.

Before release, validate registration, restart, container lifecycle, CNI metadata, DNS search, storage attach/mount/unmount, Windows service migration, and rollback against an isolated compatibility control plane.
