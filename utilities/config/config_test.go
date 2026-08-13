//go:build !windows
// +build !windows

package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/check.v1"

	"github.com/PastureStack/node-agent/utilities/constants"
	gofqdn "github.com/ShowMax/go-fqdn"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type ConfigTestSuite struct {
}

var _ = check.Suite(&ConfigTestSuite{})

func (s *ConfigTestSuite) SetUpSuite(c *check.C) {
}

func (s *ConfigTestSuite) TestLabels(c *check.C) {
	constants.ConfigOverride["HOST_LABELS"] = "foo=bar&test=1&foo=dontpick&novalue"
	labels := Labels()
	c.Assert(labels, check.DeepEquals, map[string]string{"foo": "bar", "test": "1", "novalue": ""})

}

func (s *ConfigTestSuite) TestManagedStatePathRejectsArbitraryFiles(c *check.C) {
	originalState, hadState := constants.ConfigOverride["STATE_DIR"]
	originalPhysical, hadPhysical := constants.ConfigOverride["PHYSICAL_HOST_UUID_FILE"]
	defer func() {
		if hadState {
			constants.ConfigOverride["STATE_DIR"] = originalState
		} else {
			delete(constants.ConfigOverride, "STATE_DIR")
		}
		if hadPhysical {
			constants.ConfigOverride["PHYSICAL_HOST_UUID_FILE"] = originalPhysical
		} else {
			delete(constants.ConfigOverride, "PHYSICAL_HOST_UUID_FILE")
		}
	}()

	constants.ConfigOverride["STATE_DIR"] = filepath.Join(os.TempDir(), "outside-state")
	constants.ConfigOverride["PHYSICAL_HOST_UUID_FILE"] = filepath.Join(os.TempDir(), "sensitive")
	c.Assert(StateDir(), check.Equals, defaultStateDir())
	c.Assert(physicalHostUUIDFile(), check.Equals, filepath.Join(defaultStateDir(), ".physical_host_uuid"))
	_, _, err := resolveIdentityFile(filepath.Join(os.TempDir(), "sensitive"))
	c.Assert(err, check.NotNil)
}

func (s *ConfigTestSuite) TestManagedStatePathPreservesApprovedLegacyLocation(c *check.C) {
	if runtime.GOOS == "windows" {
		c.Skip("Linux compatibility path")
	}
	original, hadValue := constants.ConfigOverride["STATE_DIR"]
	defer func() {
		if hadValue {
			constants.ConfigOverride["STATE_DIR"] = original
		} else {
			delete(constants.ConfigOverride, "STATE_DIR")
		}
	}()
	constants.ConfigOverride["STATE_DIR"] = "/var/lib/rancher/state"
	c.Assert(StateDir(), check.Equals, "/var/lib/rancher/state")
	c.Assert(dockerUUIDFile(), check.Equals, "/var/lib/rancher/state/.docker_uuid")
}

func (s *ConfigTestSuite) TestPublicKeyPathRejectsUnmanagedOverride(c *check.C) {
	original, hadValue := constants.ConfigOverride["CONSOLE_HOST_API_PUBLIC_KEY"]
	defer func() {
		if hadValue {
			constants.ConfigOverride["CONSOLE_HOST_API_PUBLIC_KEY"] = original
		} else {
			delete(constants.ConfigOverride, "CONSOLE_HOST_API_PUBLIC_KEY")
		}
	}()
	constants.ConfigOverride["CONSOLE_HOST_API_PUBLIC_KEY"] = filepath.Join(os.TempDir(), "untrusted.crt")
	preferred, _ := approvedPublicKeyPaths(Home())
	c.Assert(JwtPublicKeyFile(), check.Equals, preferred)
}

func (s *ConfigTestSuite) TestManagedExecutionPathsRejectArbitraryOverrides(c *check.C) {
	overrides := map[string]string{
		"HOME":          "/tmp/untrusted-home",
		"STATE_DIR":     "/tmp/untrusted-state",
		"BUILD_DIR":     "/tmp/untrusted-build",
		"CONFIG_SCRIPT": "/tmp/untrusted-command",
		"HOST_KEY_FILE": "/tmp/untrusted-key",
		"STAMP_FILE":    "/tmp/untrusted-stamp",
	}
	originals := map[string]string{}
	present := map[string]bool{}
	for name, value := range overrides {
		originals[name], present[name] = constants.ConfigOverride[name]
		constants.ConfigOverride[name] = value
	}
	defer func() {
		for name := range overrides {
			if present[name] {
				constants.ConfigOverride[name] = originals[name]
			} else {
				delete(constants.ConfigOverride, name)
			}
		}
	}()

	c.Assert(Home(), check.Equals, "/var/lib/pasturestack")
	c.Assert(StateDir(), check.Equals, "/var/lib/pasturestack")
	c.Assert(Builds(), check.Equals, "/var/lib/pasturestack/builds")
	c.Assert(Sh(), check.Equals, "/var/lib/pasturestack/config.sh")
	c.Assert(KeyFile(), check.Equals, "/var/lib/pasturestack/etc/ssl/host.key")
	c.Assert(Stamp(), check.Equals, "/var/lib/pasturestack/.pyagent-stamp")
}

func (s *ConfigTestSuite) TestManagedExecutionPathsPreserveLegacyLocations(c *check.C) {
	originalHome, hadHome := constants.ConfigOverride["HOME"]
	originalState, hadState := constants.ConfigOverride["STATE_DIR"]
	defer func() {
		if hadHome {
			constants.ConfigOverride["HOME"] = originalHome
		} else {
			delete(constants.ConfigOverride, "HOME")
		}
		if hadState {
			constants.ConfigOverride["STATE_DIR"] = originalState
		} else {
			delete(constants.ConfigOverride, "STATE_DIR")
		}
	}()

	constants.ConfigOverride["HOME"] = "/var/lib/cattle"
	constants.ConfigOverride["STATE_DIR"] = "/var/lib/rancher/state"
	c.Assert(Home(), check.Equals, "/var/lib/cattle")
	c.Assert(Builds(), check.Equals, "/var/lib/cattle/builds")
	c.Assert(Sh(), check.Equals, "/var/lib/cattle/config.sh")
	c.Assert(Stamp(), check.Equals, "/var/lib/cattle/.pyagent-stamp")
	c.Assert(KeyFile(), check.Equals, "/var/lib/rancher/etc/ssl/host.key")
}

func (s *ConfigTestSuite) TestManagedExecutionPathsAcceptApprovedOverrides(c *check.C) {
	overrides := map[string]string{
		"HOME":          "/var/lib/pasturestack",
		"STATE_DIR":     "/var/lib/pasturestack",
		"BUILD_DIR":     "/var/lib/cattle/builds",
		"CONFIG_SCRIPT": "/var/lib/cattle/config.sh",
		"HOST_KEY_FILE": "/var/lib/cattle/etc/ssl/host.key",
		"STAMP_FILE":    "/var/lib/cattle/.pyagent-stamp",
	}
	originals := map[string]string{}
	present := map[string]bool{}
	for name, value := range overrides {
		originals[name], present[name] = constants.ConfigOverride[name]
		constants.ConfigOverride[name] = value
	}
	defer func() {
		for name := range overrides {
			if present[name] {
				constants.ConfigOverride[name] = originals[name]
			} else {
				delete(constants.ConfigOverride, name)
			}
		}
	}()

	c.Assert(Builds(), check.Equals, "/var/lib/cattle/builds")
	c.Assert(Sh(), check.Equals, "/var/lib/cattle/config.sh")
	c.Assert(KeyFile(), check.Equals, "/var/lib/cattle/etc/ssl/host.key")
	c.Assert(Stamp(), check.Equals, "/var/lib/cattle/.pyagent-stamp")
}

func (s *ConfigTestSuite) unTestHostName(c *check.C) {
	// by default getFQDNLinux should just have the same with getFQDNByIP
	fqdn1, err := getFQDNLinux()
	if err != nil {
		c.Fatal(err)
	}
	fqdn2 := gofqdn.Get()
	c.Assert(fqdn1, check.Equals, fqdn2)
}
