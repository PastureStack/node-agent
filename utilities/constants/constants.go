package constants

import (
	"regexp"
)

const (
	ContainerNameLabel    = "io.rancher.container.name"
	PullImageLabels       = "io.rancher.container.pull_image"
	UUIDLabel             = "io.rancher.container.uuid"
	LegacyURLLabel        = "io.rancher.container.cattle_url"
	AgentIDLabel          = "io.rancher.container.agent_id"
	LegacyAgentImageLabel = "io.rancher.host.agent_image"
	PlatformIPLabel       = "io.rancher.container.ip"
	PlatformMacLabel      = "io.rancher.container.mac_address"

	TempName   = "work"
	TempPrefix = "pasturestack-temp-"
)

var ConfigOverride = make(map[string]string)
var HTTPProxyList = []string{"http_proxy", "HTTP_PROXY", "https_proxy", "HTTPS_PROXY", "no_proxy", "NO_PROXY"}
var NameRegexCompiler = regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9_.-]+$")
