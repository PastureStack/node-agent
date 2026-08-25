# Maintained compatibility modules

These three small modules preserve the control-platform event and client APIs
used by `node-agent`. Their source comes from the repository's previously
vendored upstream revisions and retains the upstream licenses and history
notices in each directory.

They are kept in-tree because the original modules are no longer maintained.
PastureStack's copies use current, canonical Logrus, WebSocket, and error
modules and are compiled and tested together with `node-agent`. New product
features must not be added here; replace a compatibility API at its call sites
when that protocol is retired.
