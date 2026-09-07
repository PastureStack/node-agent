# Node Agent v0.13.24

Add Linux host runtime and accelerator inventory through the existing host-info
collector. Report NVIDIA UUIDs and DRM/KFD device nodes with their actual groups;
do not install drivers or invoke vendor utilities.

Preserve LaunchConfig runtime and GPU DeviceRequests through Docker create and
inspect. Reject malformed device bindings and contradictory shared-memory/IPC
settings. Hardware options never implicitly enable privileged mode or host IPC.

Windows packaging remains supported; these new resource controls target Linux.
Inventory is not proof of CUDA, ROCm or media-workload compatibility. GPU count
controls visibility, not exclusive reservation or scheduling quotas.
