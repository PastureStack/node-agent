package handlers

import (
	"strconv"
	"strings"

	"context"
	engineCli "github.com/PastureStack/node-agent/internal/dockerapi/client"
	"github.com/PastureStack/node-agent/internal/dockerapi/types"
	"github.com/go-viper/mapstructure/v2"
	"github.com/pkg/errors"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

type portCheckRequest struct {
	Ports []portCheckPort `mapstructure:"ports"`
}

type portCheckPort struct {
	BindAddress string `mapstructure:"bindAddress"`
	PublicPort  int    `mapstructure:"publicPort"`
	Protocol    string `mapstructure:"protocol"`
}

type portCheckConflict struct {
	Source        string `json:"source"`
	ContainerID   string `json:"containerId,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	State         string `json:"state"`
	BindAddress   string `json:"bindAddress"`
	PublicPort    int    `json:"publicPort"`
	Protocol      string `json:"protocol"`
}

type PortCheckHandler struct {
	dockerClient portDockerClient
}

type portDockerClient interface {
	ContainerList(context.Context, types.ContainerListOptions) ([]types.Container, error)
	ContainerInspect(context.Context, string) (types.ContainerJSON, error)
}

func (h *PortCheckHandler) PortCheck(event *revents.Event, cli *client.RancherClient) error {
	if event.ReplyTo == "" {
		return nil
	}

	request := portCheckRequest{}
	if err := mapstructure.WeakDecode(event.Data, &request); err != nil {
		return errors.Wrap(err, "failed to decode host port check request")
	}
	request.Ports = validRequestedPorts(request.Ports)

	conflicts, dockerBindings, err := h.dockerConflicts(request.Ports)
	if err != nil {
		return errors.Wrap(err, "failed to inspect Docker host port bindings")
	}

	hostConflicts, hostSocketProbeSupported := hostSocketConflicts(request.Ports, dockerBindings)
	conflicts = append(conflicts, hostConflicts...)
	values := make([]interface{}, 0, len(conflicts))
	for _, conflict := range conflicts {
		values = append(values, map[string]interface{}{
			"source":        conflict.Source,
			"containerId":   conflict.ContainerID,
			"containerName": conflict.ContainerName,
			"state":         conflict.State,
			"bindAddress":   conflict.BindAddress,
			"publicPort":    conflict.PublicPort,
			"protocol":      conflict.Protocol,
		})
	}

	return reply(map[string]interface{}{
		"supported":                true,
		"hostSocketProbeSupported": hostSocketProbeSupported,
		"conflicts":                values,
	}, event, cli)
}

func (h *PortCheckHandler) dockerConflicts(requested []portCheckPort) ([]portCheckConflict, map[string]bool, error) {
	conflicts := []portCheckConflict{}
	dockerBindings := map[string]bool{}
	containers, err := h.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return nil, nil, err
	}

	for _, container := range containers {
		inspect, err := h.dockerClient.ContainerInspect(context.Background(), container.ID)
		if err != nil {
			if engineCli.IsErrNotFound(err) {
				// The container disappeared between list and inspect, so it no
				// longer owns a binding and is safe to ignore.
				continue
			}
			return nil, nil, errors.Wrapf(err, "failed to inspect container %s", container.ID)
		}
		if inspect.ContainerJSONBase == nil {
			return nil, nil, errors.Errorf("container %s has no inspection metadata", container.ID)
		}
		if inspect.HostConfig == nil {
			return nil, nil, errors.Errorf("container %s has no host configuration", container.ID)
		}
		if inspect.State == nil {
			return nil, nil, errors.Errorf("container %s has no runtime state", container.ID)
		}
		if inspect.Name == "" {
			inspect.Name = container.ID
		}
		if inspect.State.Status == "" && container.State == "" {
			return nil, nil, errors.Errorf("container %s has no runtime status", container.ID)
		}
		if len(inspect.HostConfig.PortBindings) == 0 {
			continue
		}
		state := container.State
		if inspect.State.Status != "" {
			state = inspect.State.Status
		}
		name := strings.TrimPrefix(inspect.Name, "/")
		for containerPort, bindings := range inspect.HostConfig.PortBindings {
			protocol := strings.ToLower(string(containerPort.Proto()))
			for _, binding := range bindings {
				publicPort, parseErr := strconv.Atoi(binding.HostPort)
				if parseErr != nil || publicPort < 1 || publicPort > 65535 {
					continue
				}
				bindAddress := ""
				if binding.HostIP.IsValid() {
					bindAddress = binding.HostIP.String()
				}
				bindAddress = normalizeBindAddress(bindAddress)
				if !matchesRequested(requested, bindAddress, publicPort, protocol) {
					continue
				}
				dockerBindings[portBindingKey(bindAddress, publicPort, protocol)] = true
				conflicts = append(conflicts, portCheckConflict{
					Source:        "docker",
					ContainerID:   container.ID,
					ContainerName: name,
					State:         string(state),
					BindAddress:   bindAddress,
					PublicPort:    publicPort,
					Protocol:      protocol,
				})
			}
		}
	}

	return conflicts, dockerBindings, nil
}

func validRequestedPorts(ports []portCheckPort) []portCheckPort {
	result := []portCheckPort{}
	for _, port := range ports {
		port.Protocol = strings.ToLower(strings.TrimSpace(port.Protocol))
		if port.Protocol == "" {
			port.Protocol = "tcp"
		}
		if port.PublicPort < 1 || port.PublicPort > 65535 || (port.Protocol != "tcp" && port.Protocol != "udp") {
			continue
		}
		port.BindAddress = normalizeBindAddress(port.BindAddress)
		result = append(result, port)
	}
	return result
}

func matchesRequested(requested []portCheckPort, bindAddress string, publicPort int, protocol string) bool {
	for _, port := range requested {
		if port.PublicPort == publicPort && port.Protocol == protocol && bindAddressesOverlap(port.BindAddress, bindAddress) {
			return true
		}
	}
	return false
}

func normalizeBindAddress(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if value == "" || value == "*" {
		return "0.0.0.0"
	}
	if value == "0:0:0:0:0:0:0:0" {
		return "::"
	}
	return value
}

func bindAddressesOverlap(left, right string) bool {
	left = normalizeBindAddress(left)
	right = normalizeBindAddress(right)
	if left == "0.0.0.0" || left == "::" || right == "0.0.0.0" || right == "::" {
		return true
	}
	return left == right
}

func portBindingKey(bindAddress string, publicPort int, protocol string) string {
	return normalizeBindAddress(bindAddress) + "|" + strconv.Itoa(publicPort) + "|" + strings.ToLower(protocol)
}

func explainedByDocker(bindings map[string]bool, bindAddress string, publicPort int, protocol string) bool {
	for key := range bindings {
		parts := strings.Split(key, "|")
		if len(parts) != 3 || parts[2] != protocol {
			continue
		}
		value, err := strconv.Atoi(parts[1])
		if err == nil && value == publicPort && bindAddressesOverlap(parts[0], bindAddress) {
			return true
		}
	}
	return false
}
