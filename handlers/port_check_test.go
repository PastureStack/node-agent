package handlers

import (
	"errors"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"golang.org/x/net/context"
)

type fakePortDockerClient struct {
	containers   []types.Container
	inspections  map[string]types.ContainerJSON
	inspectError map[string]error
}

func (f *fakePortDockerClient) ContainerList(context.Context, types.ContainerListOptions) ([]types.Container, error) {
	return f.containers, nil
}

func (f *fakePortDockerClient) ContainerInspect(_ context.Context, id string) (types.ContainerJSON, error) {
	if err := f.inspectError[id]; err != nil {
		return types.ContainerJSON{}, err
	}
	return f.inspections[id], nil
}

func TestBindAddressOverlap(t *testing.T) {
	if !bindAddressesOverlap("0.0.0.0", "10.0.0.5") {
		t.Fatal("IPv4 wildcard must overlap a specific address")
	}
	if !bindAddressesOverlap("::", "10.0.0.5") {
		t.Fatal("IPv6 wildcard must be conservative on dual-stack hosts")
	}
	if bindAddressesOverlap("10.0.0.5", "10.0.0.6") {
		t.Fatal("different specific addresses must not overlap")
	}
}

func TestRequestedPortsKeepProtocolsIndependent(t *testing.T) {
	requested := validRequestedPorts([]portCheckPort{{PublicPort: 2201, Protocol: "tcp"}})
	if !matchesRequested(requested, "0.0.0.0", 2201, "tcp") {
		t.Fatal("matching TCP binding was not found")
	}
	if matchesRequested(requested, "0.0.0.0", 2201, "udp") {
		t.Fatal("TCP and UDP must have independent port namespaces")
	}
}

func TestDockerConflictsIncludeStoppedContainerBindings(t *testing.T) {
	client := &fakePortDockerClient{
		containers: []types.Container{{ID: "container-1", State: "exited"}},
		inspections: map[string]types.ContainerJSON{
			"container-1": {
				ContainerJSONBase: &types.ContainerJSONBase{
					ID:    "container-1",
					Name:  "/stopped-owner",
					State: &types.ContainerState{Status: "exited"},
					HostConfig: &container.HostConfig{PortBindings: nat.PortMap{
						nat.Port("22/tcp"): []nat.PortBinding{{HostIP: "0.0.0.0", HostPort: "2201"}},
					}},
				},
			},
		},
		inspectError: map[string]error{},
	}
	handler := PortCheckHandler{dockerClient: client}
	conflicts, bindings, err := handler.dockerConflicts([]portCheckPort{{
		BindAddress: "10.0.0.5",
		PublicPort:  2201,
		Protocol:    "tcp",
	}})

	if err != nil {
		t.Fatalf("unexpected Docker inspection error: %v", err)
	}
	if len(conflicts) != 1 || conflicts[0].State != "exited" || conflicts[0].ContainerName != "stopped-owner" {
		t.Fatalf("stopped binding was not reported correctly: %#v", conflicts)
	}
	if !explainedByDocker(bindings, "::", 2201, "tcp") {
		t.Fatal("Docker binding inventory must explain the matching host socket")
	}
}

func TestDockerInspectFailureDoesNotProduceFalseAvailableResult(t *testing.T) {
	client := &fakePortDockerClient{
		containers:   []types.Container{{ID: "container-1", State: "running"}},
		inspections:  map[string]types.ContainerJSON{},
		inspectError: map[string]error{"container-1": errors.New("inspection unavailable")},
	}
	handler := PortCheckHandler{dockerClient: client}
	_, _, err := handler.dockerConflicts([]portCheckPort{{PublicPort: 2201, Protocol: "tcp"}})

	if err == nil {
		t.Fatal("an incomplete Docker inventory must return an error instead of reporting the port available")
	}
}
