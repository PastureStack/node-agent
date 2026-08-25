package docker

import (
	"fmt"
	"os"
	"time"

	dockerClient "github.com/PastureStack/node-agent/internal/dockerapi/client"
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/pkg/errors"
)

const DefaultVersion = ""

func launchDefaultClient(version string) (*dockerClient.Client, error) {
	ip := fmt.Sprintf("tcp://%v:2375", os.Getenv("DEFAULT_GATEWAY"))
	if os.Getenv("DEFAULT_GATEWAY") == "" {
		return dockerClient.New(dockerClient.FromEnv)
	}
	cliFromAgent, cerr := dockerClient.New(dockerClient.WithHost(ip), dockerClient.WithAPIVersion(version))
	if cerr != nil {
		return nil, errors.Wrap(cerr, constants.LaunchDefaultClientError)
	}
	return cliFromAgent, nil
}

func NewEnvClientWithTimeout(timeout time.Duration) (*dockerClient.Client, error) {
	if gateway := os.Getenv("DEFAULT_GATEWAY"); gateway != "" {
		return dockerClient.New(
			dockerClient.WithHost(fmt.Sprintf("tcp://%v:2375", gateway)),
			dockerClient.WithTimeout(timeout),
		)
	}
	return dockerClient.New(dockerClient.FromEnv, dockerClient.WithTimeout(timeout))
}
