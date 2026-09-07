//go:build linux

package hostinfo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHardwareDiscoveryUsesIdentityNotDeviceGuess(t *testing.T) {
	root := t.TempDir()
	write := func(name, text string) {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("sys/class/drm/renderD128/device/vendor", "0x15ad")
	write("sys/class/drm/renderD128/device/device", "0x0405")
	if err := os.MkdirAll(filepath.Join(root, "dev/dri"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/dev/null", filepath.Join(root, "dev/dri/renderD128")); err != nil {
		t.Fatal(err)
	}
	write("proc/driver/nvidia/gpus/0000:01:00.0/information", "Model: Test GPU\nGPU UUID: GPU-stable-id\n")
	devices, err := discoverHardware(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected DRM and NVIDIA devices: %+v", devices)
	}
	for _, device := range devices {
		if device["kind"] == "drm" && (device["vendor"] != "VMware" || device["path"] != "/dev/dri/renderD128" || device["groupId"] == nil) {
			t.Fatalf("wrong DRM identity: %+v", device)
		}
		if device["kind"] == "nvidia" && device["id"] != "GPU-stable-id" {
			t.Fatal("NVIDIA identity lost")
		}
	}
}

func TestHardwareDiscoveryMissingDevicesIsNotGPUCapability(t *testing.T) {
	devices, err := discoverHardware(t.TempDir())
	if err != nil || len(devices) != 0 {
		t.Fatalf("missing hardware: %v %v", devices, err)
	}
}
