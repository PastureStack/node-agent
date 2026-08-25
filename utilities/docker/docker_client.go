package docker

import (
	"github.com/PastureStack/node-agent/internal/dockerapi/client"
)

func GetClient(version string) *client.Client {
	defCli, err := launchDefaultClient(version)
	if err != nil {
		panic(err)
	}
	return defCli
}
