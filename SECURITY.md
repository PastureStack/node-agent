# Security Policy

## Supported state

This repository is under migration review and is not release-ready. Do not deploy an unreviewed worktree binary or image in production.

## Security boundaries

- Registration URLs, API keys, host UUIDs, certificates, and event payloads are sensitive.
- Docker socket, host namespaces, storage sockets, and host filesystem mounts grant elevated access.
- Compatibility environment variables and labels must not be logged with their values.
- Vendored dependencies and test images require review before any release.
- Do not commit credentials, private registry coordinates, host data, certificates, or live event fixtures.
- `Dockerfile.dapper` is a local build-and-test image, not a runtime image. Its privileged root process is limited to a disposable inner Docker daemon backed by an anonymous data volume; the harness does not mount the host Docker socket. Never publish or deploy this image as a service.

## Reporting

Report suspected vulnerabilities through this repository's private security advisory channel. Do not include live credentials or production event data in a public issue.
