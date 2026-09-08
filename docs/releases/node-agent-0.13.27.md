# Node Agent v0.13.27

This release completes Docker inspect event compatibility for storage events.

- Route volume activate and remove payloads through the same typed Docker
  decoder used by instance lifecycle events.
- Accept current Docker network address, hardware address, and empty port-set
  values when they appear below a volume's attached instance.
- Preserve the existing storage and instance API contracts while preventing a
  connected host from repeatedly rejecting queued volume reconciliation.
