package utils

import (
	"net/netip"
	"testing"

	"github.com/PastureStack/node-agent/internal/dockerapi/types"
	"github.com/PastureStack/node-agent/model"
	"github.com/moby/moby/api/types/network"
	revents "github.com/rancher/event-subscriber/events"
)

func TestIsNodeAgentContainer(t *testing.T) {
	tests := []struct {
		name      string
		names     []string
		protected bool
	}{
		{
			name:      "current PastureStack container",
			names:     []string{"/pasturestack-node-agent"},
			protected: true,
		},
		{
			name:      "legacy compatibility container",
			names:     []string{"/rancher-agent"},
			protected: true,
		},
		{
			name:      "agent alias is not first",
			names:     []string{"/unrelated-alias", "/pasturestack-node-agent"},
			protected: true,
		},
		{
			name:      "ordinary workload",
			names:     []string{"/web"},
			protected: false,
		},
		{
			name:      "unnamed container",
			names:     nil,
			protected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			container := types.Container{Names: test.names}
			if actual := IsNodeAgentContainer(container); actual != test.protected {
				t.Fatalf("IsNodeAgentContainer(%v) = %v, want %v", test.names, actual, test.protected)
			}
		})
	}
}

func TestDecodeEventModelDecodesNestedVolumeDockerInspectTypes(t *testing.T) {
	payload := map[string]interface{}{
		"volume": map[string]interface{}{
			"id": 23,
			"instance": map[string]interface{}{
				"data": map[string]interface{}{
					"dockerInspect": map[string]interface{}{
						"Config": map[string]interface{}{
							"ExposedPorts": map[string]interface{}{"8090/tcp": ""},
						},
						"NetworkSettings": map[string]interface{}{
							"Networks": map[string]interface{}{
								"host": map[string]interface{}{
									"Gateway":           "",
									"IPAddress":         "",
									"MacAddress":        "",
									"IPv6Gateway":       "",
									"GlobalIPv6Address": "",
								},
							},
						},
					},
				},
			},
		},
	}

	var decoded model.VolumeStoragePoolMap
	if err := DecodeEventModel(payload, &decoded); err != nil {
		t.Fatalf("DecodeEventModel() error = %v", err)
	}
	if decoded.Volume.ID != 23 {
		t.Fatalf("decoded volume id = %d, want 23", decoded.Volume.ID)
	}
	if _, ok := decoded.Volume.Instance.Data.DockerInspect.Config.ExposedPorts[network.MustParsePort("8090/tcp")]; !ok {
		t.Fatal("nested empty Docker port-set value was not decoded")
	}
	if endpoint := decoded.Volume.Instance.Data.DockerInspect.NetworkSettings.Networks["host"]; endpoint == nil {
		t.Fatal("nested host endpoint was not decoded")
	}
}

func TestGetInstanceAndHostDecodesCurrentDockerInspectTypes(t *testing.T) {
	dockerInspect := map[string]interface{}{
		"Config": map[string]interface{}{
			"ExposedPorts": map[string]interface{}{"8080/tcp": ""},
		},
		"NetworkSettings": map[string]interface{}{
			"Networks": map[string]interface{}{
				"bridge": map[string]interface{}{
					"Gateway":           "172.17.0.1",
					"IPAddress":         "172.17.0.2",
					"MacAddress":        "02:42:ac:11:00:02",
					"IPv6Gateway":       "",
					"GlobalIPv6Address": "",
				},
			},
		},
	}
	event := &revents.Event{Data: map[string]interface{}{
		"instanceHostMap": map[string]interface{}{
			"instance": map[string]interface{}{
				"id": 17,
				"data": map[string]interface{}{
					"dockerInspect": dockerInspect,
				},
				"volumes": []interface{}{
					map[string]interface{}{
						"instance": map[string]interface{}{
							"data": map[string]interface{}{"dockerInspect": dockerInspect},
						},
					},
				},
			},
			"host": map[string]interface{}{"id": 10, "state": "active"},
		},
	}}

	instance, host, err := GetInstanceAndHost(event)
	if err != nil {
		t.Fatalf("GetInstanceAndHost() error = %v", err)
	}
	if instance.ID != 17 || host.ID != 10 || host.State != "active" {
		t.Fatalf("decoded identities = instance %d, host %d/%q", instance.ID, host.ID, host.State)
	}
	endpoint := instance.Data.DockerInspect.NetworkSettings.Networks["bridge"]
	if endpoint == nil || endpoint.IPAddress != netip.MustParseAddr("172.17.0.2") || endpoint.MacAddress.String() != "02:42:ac:11:00:02" {
		t.Fatalf("decoded endpoint = %#v", endpoint)
	}
	if _, ok := instance.Data.DockerInspect.Config.ExposedPorts[network.MustParsePort("8080/tcp")]; !ok {
		t.Fatal("empty Docker port-set value was not decoded")
	}
	nested := instance.Volumes[0].Instance.Data.DockerInspect.NetworkSettings.Networks["bridge"]
	if nested == nil || nested.IPAddress != netip.MustParseAddr("172.17.0.2") || nested.MacAddress.String() != "02:42:ac:11:00:02" {
		t.Fatalf("decoded nested endpoint = %#v", nested)
	}
}
