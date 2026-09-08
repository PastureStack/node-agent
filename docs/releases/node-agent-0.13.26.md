# Node Agent v0.13.26

This release restores event handling against current Docker inspect payloads.

- Decode current typed Docker network addresses, hardware addresses, and port
  sets without rejecting the control-plane event representation.
- Preserve existing event models and API fields while accepting empty Docker
  set values.
- Cover the exact network and exposed-port payload that previously left a
  reconnected host online but unable to finish workload reconciliation.
