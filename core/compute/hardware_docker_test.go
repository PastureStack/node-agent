//go:build linux

package compute

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/PastureStack/node-agent/internal/dockerapi/client"
	"github.com/PastureStack/node-agent/internal/dockerapi/types"
	"github.com/PastureStack/node-agent/model"
	"github.com/moby/moby/api/types/container"
)

// Explicit opt-in: uses an already present immutable image, an isolated network
// namespace, no host mounts/ports, and removes only the exact container it made.
// Do not invoke the legacy whole-daemon test harness on a shared development VM.
func TestHardwareDockerRoundTrip(t *testing.T) {
	image := os.Getenv("PASTURESTACK_HARDWARE_TEST_IMAGE")
	if image == "" {
		t.Skip("set PASTURESTACK_HARDWARE_TEST_IMAGE to an existing approved sha256 image")
	}
	if len(image) != 71 || image[:7] != "sha256:" {
		t.Fatal("test image must be an immutable local image ID")
	}
	cli, err := client.NewEnvClient()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	var fields model.InstanceFields
	// Decode through the real agent wire contract, including signed limits.
	if err = json.Unmarshal([]byte(`{"shmSize":2147483648,"ipcMode":"private","runtime":"runc","pidsLimit":64,"cpuQuota":50000,"cpuPeriod":100000,"runInit":true,"devices":["/dev/null:rw"],"groupAdd":["993"],"tmpfs":{"/tmp":"rw,noexec,nosuid,size=64m"},"sysctls":{"net.core.somaxconn":"1024"},"ulimits":[{"name":"nofile","soft":1024,"hard":4096}]}`), &fields); err != nil {
		t.Fatal(err)
	}
	var hc container.HostConfig
	if err = setupFieldsHostConfig(fields, &hc); err != nil {
		t.Fatal(err)
	}
	hc.NetworkMode = "none"
	created, err := cli.ContainerCreate(ctx, &container.Config{Image: image, Entrypoint: []string{"/bin/sh"},
		Cmd: []string{"-c", "sleep 60"}, Labels: map[string]string{"io.pasturestack.test": "hardware-roundtrip"}}, &hc, nil,
		fmt.Sprintf("pasturestack-hardware-test-%d", time.Now().UnixNano()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, done := context.WithTimeout(context.Background(), 20*time.Second)
		defer done()
		if err := cli.ContainerRemove(cleanup, created.ID, types.ContainerRemoveOptions{Force: true}); err != nil {
			t.Errorf("exact test container cleanup: %v", err)
		}
	})
	if err = cli.ContainerStart(ctx, created.ID, types.ContainerStartOptions{}); err != nil {
		t.Fatal(err)
	}
	actual, err := cli.ContainerInspect(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !actual.State.Running {
		t.Fatal("fixture did not start")
	}
	c := actual.HostConfig
	if c.ShmSize != 2147483648 || c.Runtime != "runc" || c.IpcMode != "private" || c.PidsLimit == nil || *c.PidsLimit != 64 ||
		c.CPUQuota != 50000 || c.CPUPeriod != 100000 || c.Init == nil || !*c.Init || c.Privileged || len(c.CapAdd) != 0 || len(c.SecurityOpt) != 0 {
		t.Fatal("Docker runtime did not preserve resource/isolation settings")
	}
	if c.Tmpfs["/tmp"] != "rw,noexec,nosuid,size=64m" || c.Sysctls["net.core.somaxconn"] != "1024" ||
		len(c.Ulimits) != 1 || c.Ulimits[0].Name != "nofile" || c.Ulimits[0].Soft != 1024 || c.Ulimits[0].Hard != 4096 ||
		len(c.Devices) != 1 || c.Devices[0].PathOnHost != "/dev/null" || c.Devices[0].CgroupPermissions != "rw" ||
		len(c.GroupAdd) != 1 || c.GroupAdd[0] != "993" {
		t.Fatal("Docker runtime lost advanced options")
	}
	t.Log("real Docker create/start/inspect passed; GPU hardware execution is not covered by this test")
}
