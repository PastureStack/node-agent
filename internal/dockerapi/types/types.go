// Package types preserves the node-agent's stable Docker-facing data contract
// while the implementation uses the current modular Moby API.
package types

import (
	"io"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/registry"
	"github.com/moby/moby/api/types/system"
	mobyclient "github.com/moby/moby/client"
)

type AuthConfig = registry.AuthConfig
type Container = container.Summary
type ContainerCreateResponse = container.CreateResponse
type ContainerExecCreateResponse = container.ExecCreateResponse
type ContainerJSONBase = container.InspectResponse
type ContainerState = container.State
type ImageInspect = image.InspectResponse
type Info = system.Info
type MountPoint = container.MountPoint
type Version = system.VersionResponse

type ContainerListOptions = mobyclient.ContainerListOptions
type ContainerLogsOptions = mobyclient.ContainerLogsOptions
type ContainerRemoveOptions = mobyclient.ContainerRemoveOptions
type ContainerStartOptions = mobyclient.ContainerStartOptions
type EventsOptions = mobyclient.EventsListOptions
type ImageBuildOptions = mobyclient.ImageBuildOptions
type ImageListOptions = mobyclient.ImageListOptions
type ImagePullOptions = mobyclient.ImagePullOptions
type ImageRemoveOptions = mobyclient.ImageRemoveOptions
type ResizeOptions = mobyclient.ContainerResizeOptions
type VolumeCreateRequest = mobyclient.VolumeCreateOptions

// ContainerJSON keeps the legacy shape consumed by node-agent while every
// field is populated from the current Moby container.InspectResponse.
type ContainerJSON struct {
	*ContainerJSONBase
	Mounts          []container.MountPoint
	Config          *ContainerConfig
	NetworkSettings *NetworkSettings
}

type ContainerConfig struct {
	*container.Config
	MacAddress string `json:"MacAddress,omitempty"`
}

type NetworkSettings struct {
	*container.NetworkSettings
	IPAddress  string                               `json:"IPAddress,omitempty"`
	MacAddress string                               `json:"MacAddress,omitempty"`
	Networks   map[string]*network.EndpointSettings `json:"Networks,omitempty"`
}

type ContainerStats struct {
	Body   io.ReadCloser
	OSType string
}

type ExecConfig struct {
	User         string
	Privileged   bool
	Tty          bool
	AttachStdin  bool
	AttachStderr bool
	AttachStdout bool
	Detach       bool
	DetachKeys   string
	Env          []string
	Cmd          []string
}

type ExecStartCheck struct {
	Detach bool
	Tty    bool
}
