//go:build linux

package hostinfo

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
)

var renderNodeName = regexp.MustCompile(`^(renderD|card)[0-9]+$`)

func readHardwareText(root string, path ...string) string {
	data, err := os.ReadFile(filepath.Join(append([]string{root}, path...)...))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func hardwareNode(root, path string) (map[string]interface{}, bool) {
	info, err := os.Stat(filepath.Join(root, path))
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return nil, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil, false
	}
	return map[string]interface{}{"path": path, "groupId": stat.Gid}, true
}

func discoverHardware(root string) ([]map[string]interface{}, error) {
	devices := []map[string]interface{}{}
	entries, err := os.ReadDir(filepath.Join(root, "sys/class/drm"))
	if err != nil && !os.IsNotExist(err) {
		return devices, err
	}
	for _, entry := range entries {
		if !renderNodeName.MatchString(entry.Name()) {
			continue
		}
		node, ok := hardwareNode(root, "/dev/dri/"+entry.Name())
		if !ok {
			continue
		}
		base := "sys/class/drm/" + entry.Name() + "/device"
		vendorID := readHardwareText(root, base, "vendor")
		vendor := map[string]string{"0x8086": "Intel", "0x1002": "AMD", "0x10de": "NVIDIA", "0x15ad": "VMware"}[vendorID]
		if vendor == "" {
			vendor = "GPU"
		}
		pci, _ := filepath.EvalSymlinks(filepath.Join(root, base))
		driver, _ := os.Readlink(filepath.Join(root, base, "driver"))
		node["kind"], node["vendor"], node["vendorId"] = "drm", vendor, vendorID
		if pci != "" {
			node["pciAddress"] = filepath.Base(pci)
		}
		if driver != "" {
			node["driver"] = filepath.Base(driver)
		}
		node["name"] = fmt.Sprintf("%s %s · %s", vendor, readHardwareText(root, base, "device"), entry.Name())
		devices = append(devices, node)
	}
	if node, ok := hardwareNode(root, "/dev/kfd"); ok {
		node["kind"], node["vendor"], node["name"] = "kfd", "AMD", "AMD ROCm /dev/kfd"
		devices = append(devices, node)
	}
	// NVIDIA's kernel driver provides stable UUIDs without requiring nvidia-smi
	// to be installed in the agent image. The runtime is reported separately.
	gpus, gpuErr := os.ReadDir(filepath.Join(root, "proc/driver/nvidia/gpus"))
	if gpuErr != nil && !os.IsNotExist(gpuErr) {
		return devices, gpuErr
	}
	for _, gpu := range gpus {
		if !gpu.IsDir() {
			continue
		}
		values := map[string]string{}
		for _, line := range strings.Split(readHardwareText(root, "proc/driver/nvidia/gpus", gpu.Name(), "information"), "\n") {
			pair := strings.SplitN(line, ":", 2)
			if len(pair) == 2 {
				values[strings.TrimSpace(pair[0])] = strings.TrimSpace(pair[1])
			}
		}
		if !strings.HasPrefix(values["GPU UUID"], "GPU-") {
			continue
		}
		devices = append(devices, map[string]interface{}{
			"kind": "nvidia", "vendor": "NVIDIA", "name": values["Model"],
			"id": values["GPU UUID"], "pciAddress": gpu.Name(),
		})
	}
	sort.Slice(devices, func(i, j int) bool { return fmt.Sprint(devices[i]["name"]) < fmt.Sprint(devices[j]["name"]) })
	return devices, nil
}
