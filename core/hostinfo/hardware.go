package hostinfo

import (
	"context"
	"sort"
	"time"

	"github.com/moby/moby/api/types/system"
)

type hardwareDockerClient interface {
	Info(context.Context) (system.Info, error)
}

// HardwareCollector reports capabilities, not reservations or GPU health.
// It does not execute vendor utilities or modify the host configuration.
type HardwareCollector struct {
	DockerClient hardwareDockerClient
}

func (h HardwareCollector) KeyName() string { return "hardwareInfo" }
func (h HardwareCollector) GetLabels(string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (h HardwareCollector) GetData() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	info, err := h.DockerClient.Info(ctx)
	if err != nil {
		return map[string]interface{}{"status": "unavailable"}, err
	}
	runtimes := make([]string, 0, len(info.Runtimes))
	for name := range info.Runtimes {
		runtimes = append(runtimes, name)
	}
	sort.Strings(runtimes)
	devices, scanErr := discoverHardware("/")
	status := "available"
	if scanErr != nil {
		status = "partial"
	}
	return map[string]interface{}{
		"status": status, "collectedAt": time.Now().UTC().Format(time.RFC3339),
		"runtimes": runtimes, "defaultRuntime": info.DefaultRuntime,
		"devices": devices, "deviceRequestsSupported": true,
	}, scanErr
}
