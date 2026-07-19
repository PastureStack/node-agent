package config

import (
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/pkg/errors"
	"os"
)

func Hostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", errors.Wrap(err, constants.HostNameError)
	}
	return DefaultValue("HOSTNAME", hostname), nil
}
