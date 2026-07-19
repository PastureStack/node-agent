//go:build !windows
// +build !windows

package events

import (
	"github.com/docker/docker/client"
)

const (
	defaultAPIVersion = ""
)

func NewDockerClient() (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	cli.UpdateClientVersion(defaultAPIVersion)
	return cli, nil
}
