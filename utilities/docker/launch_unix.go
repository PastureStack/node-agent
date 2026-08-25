//go:build linux || freebsd || solaris || openbsd || darwin
// +build linux freebsd solaris openbsd darwin

package docker

import (
	"sync"
	"time"

	dclient "github.com/PastureStack/node-agent/internal/dockerapi/client"
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/pkg/errors"
)

var (
	versionClients = map[string]*dclient.Client{}
	timeoutClients = map[time.Duration]*dclient.Client{}
	clientLock     = sync.Mutex{}
)

const DefaultVersion = ""

func launchDefaultClient(version string) (*dclient.Client, error) {
	clientLock.Lock()
	defer clientLock.Unlock()

	if c, ok := versionClients[version]; ok {
		return c, nil
	}

	opts := []dclient.Opt{dclient.FromEnv}
	if version != "" {
		opts = append(opts, dclient.WithAPIVersion(version))
	}
	cli, err := dclient.New(opts...)
	if err != nil {
		return nil, errors.Wrap(err, constants.LaunchDefaultClientError)
	}

	versionClients[version] = cli
	return cli, nil
}

func NewEnvClientWithTimeout(timeout time.Duration) (*dclient.Client, error) {
	clientLock.Lock()
	defer clientLock.Unlock()

	if c, ok := timeoutClients[timeout]; ok {
		return c, nil
	}

	c, err := dclient.New(dclient.FromEnv, dclient.WithTimeout(timeout))
	if err != nil {
		return nil, err
	}

	timeoutClients[timeout] = c
	return c, nil
}
