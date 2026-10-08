//go:build linux

package events

import (
	"context"
	"path/filepath"

	"github.com/PastureStack/node-agent/service/hostapilauncher"
	"github.com/PastureStack/node-agent/utilities/config"
	"github.com/rancher/log"
)

func startHostAPI(ctx context.Context) {
	uuid, err := config.DockerUUID()
	if err != nil || uuid == "" {
		log.Error("Host API producer identity is unavailable")
		return
	}
	home := config.Home()
	settings := hostapilauncher.Settings{
		Binary: filepath.Join(home, "bin", "host-api"),
		Values: map[string]string{
			"HOST_API_AUTH":                 "true",
			"HOST_API_HOST_UUID_CHECK":      "true",
			"HOST_API_HOST_UUID":            uuid,
			"HOST_API_PUBLIC_KEY":           config.JwtPublicKeyFile(),
			"HOST_API_DOCKER_HOST":          "unix:///var/run/docker.sock",
			"HOST_API_IP":                   config.HostAPIIP(),
			"HOST_API_PORT":                 config.HostAPIPort(),
			"HOST_API_PLATFORM_URL":         config.APIURL(""),
			"HOST_API_PLATFORM_ACCESS_KEY":  config.AccessKey(),
			"HOST_API_PLATFORM_SECRET_KEY":  config.SecretKey(),
			"HOST_API_COMPLETION_SPOOL_DIR": filepath.Join(config.StateDir(), "host-api", "completion-spool"),
			"PASTURESTACK_HOME":             home,
			"PASTURESTACK_STATE_DIR":        config.StateDir(),
			"PASTURESTACK_LOCALE":           config.DefaultValue("LOCALE", "en-US"),
		},
	}
	// The authenticated configcontent installer puts the separately maintained
	// host-api producer at this fixed path. Node owns lifecycle, not delegation.
	hostapilauncher.Run(ctx, settings)
}
