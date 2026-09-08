# Node Agent v0.13.25

This release corrects the Linux compatibility archive consumed by existing
PastureStack host installers.

- Keep the legacy `SHA1SUMS` and `SHA1SUMSSUM` files for older hosts.
- Add `SHA256SUMS` and `SHA256SUMSSUM` for current hosts.
- Verify both checksum chains against the extracted release archive in CI.

Runtime hardware inventory and Docker request behavior are unchanged from
v0.13.24.
