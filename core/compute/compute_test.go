//go:build !windows
// +build !windows

package compute

import (
	"fmt"
	"os"
	"testing"
	"time"

	"context"
	"github.com/PastureStack/node-agent/internal/dockerapi/types"
	"github.com/PastureStack/node-agent/model"
	"github.com/PastureStack/node-agent/utilities/config"
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/PastureStack/node-agent/utilities/docker"
	"github.com/PastureStack/node-agent/utilities/utils"
	"github.com/google/uuid"
	"github.com/moby/moby/api/types/blkiodev"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/rancher/log"
	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) {
	check.TestingT(t)
}

type ComputeTestSuite struct {
}

var _ = check.Suite(&ComputeTestSuite{})

func (s *ComputeTestSuite) SetUpSuite(c *check.C) {
}

/*
// Load the event to a byte array from the specified file
rawEvent := loadEvent("../test_events/instance_activate", c)

// Optional: you can unmarshal, modify, and marshal the event data if you need to. This is equivalent to the "pre"
// functions in python-agent
event := unmarshalEvent(rawEvent, c)
instance := getInstance(event, c)
event["replyTo"] = "new-reply-to"
instance["name"] = "new-name"
rawEvent = marshalEvent(event, c)

// Run the event through the framework
reply := testEvent(rawEvent, c)

// Assert whatever you need to on the reply event. This is equivalent to the "post" functions in python-agent
c.Assert(reply.Name, check.Equals, "new-reply-to")
// As an example, once you implement some more logic, you could verify that the reply has the instance name as "new-name"
*/
func (s *ComputeTestSuite) TestMultiNicsPickMac(c *check.C) {
	instance := model.Instance{}
	instance.Nics = []model.Nic{
		{
			MacAddress:   "02:03:04:05:06:07",
			DeviceNumber: 0,
		},
		{
			MacAddress:   "02:03:04:05:06:09",
			DeviceNumber: 1,
		},
	}
	config := container.Config{Labels: map[string]string{}}
	setupMacAndIP(instance, &config, true, true)
	c.Assert(config.Labels, check.DeepEquals, map[string]string{constants.PlatformMacLabel: "02:03:04:05:06:07"})
	networkConfig := network.NetworkingConfig{}
	c.Assert(setupNetworkingConfig(&networkConfig, instance), check.IsNil)
	c.Assert(networkConfig.EndpointsConfig["bridge"].MacAddress.String(), check.Equals, "02:03:04:05:06:07")
}

func (s *ComputeTestSuite) TestSetupPortsUsesCurrentMobyTypes(c *check.C) {
	instance := model.Instance{Ports: []model.Port{{
		PrivatePort: 8080,
		PublicPort:  18080,
		Protocol:    "tcp",
		Data: model.PortData{Fields: model.PortFields{
			BindAddress: "127.0.0.1",
		}},
	}}}
	config := container.Config{}
	hostConfig := container.HostConfig{}

	c.Assert(setupPorts(&config, instance, &hostConfig), check.IsNil)
	port := network.MustParsePort("8080/tcp")
	_, exposed := config.ExposedPorts[port]
	c.Assert(exposed, check.Equals, true)
	c.Assert(hostConfig.PortBindings[port], check.HasLen, 1)
	c.Assert(hostConfig.PortBindings[port][0].HostIP.String(), check.Equals, "127.0.0.1")
	c.Assert(hostConfig.PortBindings[port][0].HostPort, check.Equals, "18080")
}

func (s *ComputeTestSuite) TestSetupPortsRejectsInvalidBindAddress(c *check.C) {
	instance := model.Instance{Ports: []model.Port{{
		PrivatePort: 8080,
		Protocol:    "tcp",
		Data: model.PortData{Fields: model.PortFields{
			BindAddress: "not-an-ip-address",
		}},
	}}}
	c.Assert(setupPorts(&container.Config{}, instance, &container.HostConfig{}), check.NotNil)
}

func (s *ComputeTestSuite) TestSetupFieldsHostConfigUsesCurrentMobyTypes(c *check.C) {
	fields := model.InstanceFields{
		DNS:          []string{"1.1.1.1"},
		PidsLimit:    64,
		KernelMemory: 1024,
		DiskQuota:    2048,
	}
	hostConfig := container.HostConfig{}
	c.Assert(setupFieldsHostConfig(fields, &hostConfig), check.IsNil)
	c.Assert(hostConfig.DNS, check.HasLen, 1)
	c.Assert(hostConfig.DNS[0].String(), check.Equals, "1.1.1.1")
	c.Assert(hostConfig.PidsLimit, check.NotNil)
	c.Assert(*hostConfig.PidsLimit, check.Equals, int64(64))

	fields.DNS = []string{"not-an-ip-address"}
	c.Assert(setupFieldsHostConfig(fields, &container.HostConfig{}), check.NotNil)
}

func (s *ComputeTestSuite) TestDefaultDisk(c *check.C) {
	device := "/dev/mock"
	instance := model.Instance{}
	instance.Data = model.InstanceFieldsData{
		Fields: model.InstanceFields{
			BlkioDeviceOptions: map[string]model.DeviceOptions{
				"DEFAULT_DISK": {
					ReadIops: 10,
				},
			},
		},
	}
	hostConfig := container.HostConfig{}
	setupDeviceOptionsTest(&hostConfig, &instance, device)
	c.Assert(*hostConfig.BlkioDeviceReadIOps[0], check.DeepEquals, blkiodev.ThrottleDevice{Path: device, Rate: 10})
	hostConfig = container.HostConfig{}
	device = ""
	setupDeviceOptionsTest(&hostConfig, &instance, "")
	c.Assert(hostConfig.BlkioDeviceReadIOps, check.HasLen, 0)
}

func (s *ComputeTestSuite) TestNoLabelField(c *check.C) {
	deleteContainer("/no-label-test")
	client := docker.GetClient(docker.DefaultVersion)
	config := container.Config{Image: "ibuildthecloud/helloworld:latest"}
	resp, err := client.ContainerCreate(context.Background(), &config, nil, nil, "no-label-test")
	if err != nil {
		c.Fatal(err)
	}
	err = client.ContainerStart(context.Background(), resp.ID, types.ContainerStartOptions{})
	if err != nil {
		c.Fatal(err)
	}
	instance := model.Instance{
		UUID:       "irrelevant",
		ExternalID: resp.ID,
	}
	container, err := utils.GetContainer(client, instance, false)
	if err != nil {
		c.Fatal(err)
	}
	containers := []model.PingResource{}

	container.Labels = map[string]string{}
	containers = utils.AddContainer("running", container, containers, client)
	c.Assert(containers[0].UUID, check.Equals, "no-label-test")
	c.Assert(containers[0].SystemContainer, check.Equals, "")

	containers = []model.PingResource{}
	container.Labels = map[string]string{}
	containers = utils.AddContainer("running", container, containers, client)
	c.Assert(containers[0].UUID, check.Equals, "no-label-test")
	c.Assert(containers[0].SystemContainer, check.Equals, "")
}

func (s *ComputeTestSuite) TestDefaultValue(c *check.C) {
	varName, _ := uuid.NewRandom()
	legacyVarName := fmt.Sprintf("CATTLE_%v", varName)
	def := "defaulted"
	actual := config.DefaultValue(varName.String(), def)
	c.Assert(def, check.Equals, actual)

	actual = config.DefaultValue(varName.String(), "")
	c.Assert(actual, check.Equals, "")

	os.Setenv(legacyVarName, "")
	actual = config.DefaultValue(varName.String(), def)
	c.Assert(actual, check.Equals, def)

	os.Setenv(legacyVarName, "foobar")
	actual = config.DefaultValue(varName.String(), def)
	c.Assert(actual, check.Equals, "foobar")

	config.SetSecretKey("supersecretkey")
	actual = config.DefaultValue("SECRET_KEY", def)
	c.Assert(actual, check.Equals, "supersecretkey")
}

func (s *ComputeTestSuite) TestSetupProxy(c *check.C) {
	instance := model.Instance{System: true}

	// test case 1: test override by host_entries
	config1 := &container.Config{Env: []string{"no_proxy=bar"}}
	setupProxy(instance, config1, map[string]string{"no_proxy": "bar1"})
	c.Assert(config1.Env, check.DeepEquals, []string{"no_proxy=bar1"})

	// test case 2: ignore empty no_proxy value
	config2 := &container.Config{Env: []string{}}
	setupProxy(instance, config2, map[string]string{"no_proxy": ""})
	c.Assert(config2.Env, check.DeepEquals, []string{})

	// test case 3: no-equal case
	config3 := &container.Config{Env: []string{"no_proxy=foo"}}
	setupProxy(instance, config3, map[string]string{})
	c.Assert(config3.Env, check.DeepEquals, []string{"no_proxy=foo"})

	// test case 4: normal case
	config4 := &container.Config{Env: []string{}}
	setupProxy(instance, config4, map[string]string{"no_proxy": "foo"})
	c.Assert(config4.Env, check.DeepEquals, []string{"no_proxy=foo"})

	// test case 5: override no-equal case
	config5 := &container.Config{Env: []string{"no_proxy"}}
	setupProxy(instance, config5, map[string]string{"no_proxy": "foo"})
	c.Assert(config5.Env, check.DeepEquals, []string{"no_proxy=foo"})

	// test case 6: override non-setting
	config6 := &container.Config{Env: []string{"no_proxy", "http_proxy=", "https_proxy=foo"}}
	setupProxy(instance, config6, map[string]string{})
	c.Assert(utils.SearchInList(config6.Env, "no_proxy"), check.Equals, true)
	c.Assert(utils.SearchInList(config6.Env, "http_proxy="), check.Equals, true)
	c.Assert(utils.SearchInList(config6.Env, "https_proxy=foo"), check.Equals, true)
}

func setupDeviceOptionsTest(hostConfig *container.HostConfig, instance *model.Instance, mockDevice string) {
	deviceOptions := instance.Data.Fields.BlkioDeviceOptions

	blkioWeightDevice := []*blkiodev.WeightDevice{}
	blkioDeviceReadIOps := []*blkiodev.ThrottleDevice{}
	blkioDeviceWriteBps := []*blkiodev.ThrottleDevice{}
	blkioDeviceReadBps := []*blkiodev.ThrottleDevice{}
	blkioDeviceWriteIOps := []*blkiodev.ThrottleDevice{}

	for dev, options := range deviceOptions {
		if dev == "DEFAULT_DISK" {
			dev = mockDevice
			if dev == "" {
				log.Warnf("Couldn't find default device. Not setting device options: %+v", options)
				continue
			}
		}
		blkioWeightDevice = append(blkioWeightDevice, &blkiodev.WeightDevice{
			Path:   dev,
			Weight: options.Weight,
		})
		blkioDeviceReadIOps = append(blkioDeviceReadIOps, &blkiodev.ThrottleDevice{
			Path: dev,
			Rate: options.ReadIops,
		})
		blkioDeviceWriteIOps = append(blkioDeviceWriteIOps, &blkiodev.ThrottleDevice{
			Path: dev,
			Rate: options.WriteIops,
		})
		blkioDeviceReadBps = append(blkioDeviceReadBps, &blkiodev.ThrottleDevice{
			Path: dev,
			Rate: options.ReadBps,
		})
		blkioDeviceWriteBps = append(blkioDeviceWriteBps, &blkiodev.ThrottleDevice{
			Path: dev,
			Rate: options.WriteBps,
		})
	}
	if len(blkioWeightDevice) > 0 {
		hostConfig.BlkioWeightDevice = blkioWeightDevice
	}
	if len(blkioDeviceReadIOps) > 0 {
		hostConfig.BlkioDeviceReadIOps = blkioDeviceReadIOps
	}
	if len(blkioDeviceWriteIOps) > 0 {
		hostConfig.BlkioDeviceWriteIOps = blkioDeviceWriteIOps
	}
	if len(blkioDeviceReadBps) > 0 {
		hostConfig.BlkioDeviceReadBps = blkioDeviceReadBps
	}
	if len(blkioDeviceWriteBps) > 0 {
		hostConfig.BlkioDeviceWriteBps = blkioDeviceWriteBps
	}
}

func deleteContainer(name string) {
	client := docker.GetClient(docker.DefaultVersion)
	containerList, _ := client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	for _, c := range containerList {
		found := false
		labels := c.Labels
		if labels["io.rancher.container.uuid"] == name[1:] {
			found = true
		}

		for _, cname := range c.Names {
			if name == cname {
				found = true
				break
			}
		}
		if found {
			client.ContainerKill(context.Background(), c.ID, "KILL")
			for i := 0; i < 10; i++ {
				if inspect, err := client.ContainerInspect(context.Background(), c.ID); err == nil && inspect.State.Pid == 0 {
					break
				}
				time.Sleep(time.Duration(500) * time.Millisecond)
			}
			client.ContainerRemove(context.Background(), c.ID, types.ContainerRemoveOptions{})
		}
	}
}
