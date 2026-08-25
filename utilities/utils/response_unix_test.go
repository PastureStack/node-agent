//go:build !windows
// +build !windows

package utils

import (
	"context"
	"github.com/PastureStack/node-agent/internal/dockerapi/types"
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/PastureStack/node-agent/utilities/docker"
	"github.com/moby/moby/api/types/container"
	"github.com/patrickmn/go-cache"
	"gopkg.in/check.v1"
	"testing"
	"time"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type UtilTestSuite struct {
}

var _ = check.Suite(&UtilTestSuite{})

func (s *UtilTestSuite) SetUpSuite(c *check.C) {
}

func (s *UtilTestSuite) TestGetIP(c *check.C) {
	client := docker.GetClient(docker.DefaultVersion)
	config := container.Config{
		Image: "busybox:1",
		Cmd:   []string{"sleep", "60"},
		Labels: map[string]string{
			constants.UUIDLabel: "c861f990-4472-4fa1-960f-65171b544c29",
			cniLabels:           "true",
		},
	}

	resp, err := client.ContainerCreate(context.Background(), &config, nil, nil, "iptest")
	if err != nil {
		c.Fatal(err)
	}
	defer client.ContainerRemove(context.Background(), resp.ID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})

	err = client.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		c.Fatal(err)
	}

	inspect, err := client.ContainerInspect(context.Background(), resp.ID)
	if err != nil {
		c.Fatal(err)
	}
	cache := cache.New(5*time.Minute, 30*time.Second)
	ip, err := getIP(inspect, cache)
	if err != nil {
		c.Fatal(err)
	}
	c.Assert(ip, check.Equals, getDockerNetworkIP(inspect))
}
