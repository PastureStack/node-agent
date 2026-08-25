//go:build !windows
// +build !windows

package events

import (
	"github.com/PastureStack/node-agent/internal/dockerapi/client"
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
