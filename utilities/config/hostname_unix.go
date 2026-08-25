//go:build linux || freebsd || solaris || openbsd || darwin
// +build linux freebsd solaris openbsd darwin

package config

import (
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/pkg/errors"
)

func Hostname() (string, error) {
	hostname, err := getFQDNLinux()
	if err != nil {
		return "", errors.Wrap(err, constants.HostNameError)
	}
	return DefaultValue("HOSTNAME", hostname), nil
}

func getFQDNLinux() (string, error) {
	cmd := exec.Command("/bin/hostname", "-f")
	output, err := cmd.CombinedOutput()
	if err == nil {
		if fqdn := strings.TrimSpace(string(output)); fqdn != "" {
			return fqdn, nil
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	if fqdn, lookupErr := net.LookupCNAME(hostname); lookupErr == nil {
		if fqdn = strings.TrimSuffix(strings.TrimSpace(fqdn), "."); fqdn != "" {
			return fqdn, nil
		}
	}
	return hostname, nil
}
