//go:build linux

package compute

import (
	"encoding/json"
	"testing"

	"github.com/PastureStack/node-agent/model"
	"github.com/moby/moby/api/types/container"
)

func TestHardwareLaunchContract(t *testing.T) {
	var fields model.InstanceFields
	err := json.Unmarshal([]byte(`{"runtime":"nvidia","shmSize":2147483648,"ipcMode":"private","groupAdd":["993"],"devices":["/dev/dri/renderD128"],"deviceRequests":[{"driver":"nvidia","deviceIds":["GPU-test"],"capabilities":[["gpu"]]}]}`), &fields)
	if err != nil {
		t.Fatal(err)
	}
	var actual container.HostConfig
	if err = setupFieldsHostConfig(fields, &actual); err != nil {
		t.Fatal(err)
	}
	if actual.ShmSize != 2147483648 || actual.Runtime != "nvidia" || actual.IpcMode != "private" || actual.GroupAdd[0] != "993" {
		t.Fatalf("lost resource options: %+v", actual)
	}
	if len(actual.DeviceRequests) != 1 || actual.DeviceRequests[0].DeviceIDs[0] != "GPU-test" || actual.DeviceRequests[0].Capabilities[0][0] != "gpu" {
		t.Fatal("lost GPU request")
	}
	if actual.Devices[0].PathInContainer != "/dev/dri/renderD128" {
		t.Fatal("shorthand device not preserved")
	}
	if actual.Privileged || len(actual.SecurityOpt) > 0 {
		t.Fatal("hardware must not escalate privilege")
	}
	encoded, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	var inspected model.HostConfig
	if err := json.Unmarshal(encoded, &inspected); err != nil {
		t.Fatal(err)
	}
	if inspected.Runtime != actual.Runtime || len(inspected.DeviceRequests) != 1 || inspected.DeviceRequests[0].DeviceIDs[0] != "GPU-test" {
		t.Fatal("Docker inspect drops GPU/runtime identity")
	}
}

func TestHardwareRejectsInvalidRequests(t *testing.T) {
	cases := []model.InstanceFields{
		{ShmSize: -1}, {ShmSize: 1024, IpcMode: "host"}, {Runtime: "runc --evil"},
		{Devices: []string{""}}, {Devices: []string{"relative"}}, {Devices: []string{"/dev/dri:"}},
		{Devices: []string{"/dev/dri:/dev/dri:bad"}}, {Devices: []string{"nvidia.com/gpu=all"}},
		{DeviceRequests: []model.DeviceRequest{{Count: -2, Capabilities: [][]string{{"gpu"}}}}},
		{DeviceRequests: []model.DeviceRequest{{Count: 1, DeviceIDs: []string{"GPU-test"}, Capabilities: [][]string{{"gpu"}}}}},
		{DeviceRequests: []model.DeviceRequest{{Count: 1}}},
		{DeviceRequests: []model.DeviceRequest{{Count: 1, Capabilities: [][]string{{}}}}},
		{DeviceRequests: []model.DeviceRequest{{DeviceIDs: []string{"x", "x"}, Capabilities: [][]string{{"gpu"}}}}},
	}
	for i, fields := range cases {
		if setupFieldsHostConfig(fields, &container.HostConfig{}) == nil {
			t.Errorf("case %d accepted", i)
		}
	}
	for _, spec := range []string{"/dev/dri", "/dev/dri:rw", "/dev/dri:/dev/dri:rw"} {
		if err := setupFieldsHostConfig(model.InstanceFields{Devices: []string{spec}}, &container.HostConfig{}); err != nil {
			t.Errorf("%s: %v", spec, err)
		}
	}
}
