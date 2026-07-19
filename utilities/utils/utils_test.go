package utils

import (
	"testing"

	"github.com/docker/docker/api/types"
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
