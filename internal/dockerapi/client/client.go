// Package client adapts node-agent's established Docker operations to the
// maintained modular Moby client without leaking compatibility code through
// the product's business logic.
package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	compat "github.com/PastureStack/node-agent/internal/dockerapi/types"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/api/types/volume"
	moby "github.com/moby/moby/client"
)

type Opt = moby.Opt
type Filters = moby.Filters
type ContainerListOptions = moby.ContainerListOptions
type VolumeListOptions = moby.VolumeListOptions

var FromEnv = moby.FromEnv

func WithAPIVersion(version string) Opt { return moby.WithAPIVersion(version) }
func WithHost(host string) Opt          { return moby.WithHost(host) }
func WithTimeout(timeout time.Duration) Opt {
	return moby.WithTimeout(timeout)
}

type Client struct {
	*moby.Client
}

func New(opts ...Opt) (*Client, error) {
	inner, err := moby.New(opts...)
	if err != nil {
		return nil, err
	}
	return &Client{Client: inner}, nil
}

func NewEnvClient() (*Client, error) {
	return New(moby.FromEnv)
}

func NewClient(host, version string, httpClient *http.Client, _ map[string]string) (*Client, error) {
	opts := []moby.Opt{moby.WithHost(host)}
	if version != "" {
		opts = append(opts, moby.WithAPIVersion(version))
	}
	if httpClient != nil {
		opts = append(opts, moby.WithHTTPClient(httpClient))
	}
	return New(opts...)
}

// UpdateClientVersion is retained only for the two legacy constructors that
// immediately pin an API version after reading Docker environment settings.
func (c *Client) UpdateClientVersion(version string) {
	if version == "" {
		return
	}
	if replacement, err := moby.New(moby.FromEnv, moby.WithAPIVersion(version)); err == nil {
		_ = c.Client.Close()
		c.Client = replacement
	}
}

func (c *Client) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, name string) (compat.ContainerCreateResponse, error) {
	result, err := c.Client.ContainerCreate(ctx, moby.ContainerCreateOptions{
		Config:           config,
		HostConfig:       hostConfig,
		NetworkingConfig: networkingConfig,
		Name:             name,
	})
	if err != nil {
		return compat.ContainerCreateResponse{}, err
	}
	return compat.ContainerCreateResponse{ID: result.ID, Warnings: result.Warnings}, nil
}

func (c *Client) ContainerInspect(ctx context.Context, id string) (compat.ContainerJSON, error) {
	result, err := c.Client.ContainerInspect(ctx, id, moby.ContainerInspectOptions{})
	if err != nil {
		return compat.ContainerJSON{}, err
	}
	base := result.Container
	var config *compat.ContainerConfig
	if base.Config != nil {
		config = &compat.ContainerConfig{Config: base.Config}
	}
	var networkSettings *compat.NetworkSettings
	if base.NetworkSettings != nil {
		networkSettings = &compat.NetworkSettings{
			NetworkSettings: base.NetworkSettings,
			Networks:        base.NetworkSettings.Networks,
		}
		for _, name := range []string{"bridge", "host"} {
			if endpoint := base.NetworkSettings.Networks[name]; endpoint != nil {
				setLegacyEndpoint(networkSettings, endpoint)
				break
			}
		}
		if networkSettings.IPAddress == "" {
			for _, endpoint := range base.NetworkSettings.Networks {
				if endpoint != nil {
					setLegacyEndpoint(networkSettings, endpoint)
					break
				}
			}
		}
	}
	return compat.ContainerJSON{
		ContainerJSONBase: &base,
		Mounts:            base.Mounts,
		Config:            config,
		NetworkSettings:   networkSettings,
	}, nil
}

func setLegacyEndpoint(settings *compat.NetworkSettings, endpoint *network.EndpointSettings) {
	if endpoint.IPAddress.IsValid() {
		settings.IPAddress = endpoint.IPAddress.String()
	}
	if len(endpoint.MacAddress) != 0 {
		settings.MacAddress = endpoint.MacAddress.String()
	}
}

func (c *Client) ContainerList(ctx context.Context, options compat.ContainerListOptions) ([]compat.Container, error) {
	result, err := c.Client.ContainerList(ctx, options)
	return result.Items, err
}

func (c *Client) ContainerStart(ctx context.Context, id string, options compat.ContainerStartOptions) error {
	_, err := c.Client.ContainerStart(ctx, id, options)
	return err
}

func (c *Client) ContainerRemove(ctx context.Context, id string, options compat.ContainerRemoveOptions) error {
	_, err := c.Client.ContainerRemove(ctx, id, options)
	return err
}

func (c *Client) ContainerKill(ctx context.Context, id, signal string) error {
	_, err := c.Client.ContainerKill(ctx, id, moby.ContainerKillOptions{Signal: signal})
	return err
}

func (c *Client) ContainerPause(ctx context.Context, id string) error {
	_, err := c.Client.ContainerPause(ctx, id, moby.ContainerPauseOptions{})
	return err
}

func (c *Client) ContainerUnpause(ctx context.Context, id string) error {
	_, err := c.Client.ContainerUnpause(ctx, id, moby.ContainerUnpauseOptions{})
	return err
}

func (c *Client) ContainerStop(ctx context.Context, id string, timeout *time.Duration) error {
	var seconds *int
	if timeout != nil {
		value := int(timeout.Seconds())
		seconds = &value
	}
	_, err := c.Client.ContainerStop(ctx, id, moby.ContainerStopOptions{Timeout: seconds})
	return err
}

func (c *Client) ContainerStats(ctx context.Context, id string, stream bool) (compat.ContainerStats, error) {
	result, err := c.Client.ContainerStats(ctx, id, moby.ContainerStatsOptions{Stream: stream})
	return compat.ContainerStats{Body: result.Body}, err
}

func (c *Client) ContainerLogs(ctx context.Context, id string, options compat.ContainerLogsOptions) (io.ReadCloser, error) {
	return c.Client.ContainerLogs(ctx, id, options)
}

func (c *Client) ContainerExecCreate(ctx context.Context, id string, config compat.ExecConfig) (compat.ContainerExecCreateResponse, error) {
	result, err := c.Client.ExecCreate(ctx, id, moby.ExecCreateOptions{
		User:         config.User,
		Privileged:   config.Privileged,
		TTY:          config.Tty,
		AttachStdin:  config.AttachStdin,
		AttachStderr: config.AttachStderr,
		AttachStdout: config.AttachStdout,
		DetachKeys:   config.DetachKeys,
		Env:          config.Env,
		Cmd:          config.Cmd,
	})
	if err != nil {
		return compat.ContainerExecCreateResponse{}, err
	}
	return compat.ContainerExecCreateResponse{ID: result.ID}, nil
}

func (c *Client) ContainerExecAttach(ctx context.Context, id string, config compat.ExecConfig) (moby.HijackedResponse, error) {
	result, err := c.Client.ExecAttach(ctx, id, moby.ExecAttachOptions{TTY: config.Tty})
	return result.HijackedResponse, err
}

func (c *Client) ContainerExecStart(ctx context.Context, id string, options compat.ExecStartCheck) error {
	_, err := c.Client.ExecStart(ctx, id, moby.ExecStartOptions{Detach: options.Detach, TTY: options.Tty})
	return err
}

func (c *Client) ContainerExecResize(ctx context.Context, id string, options compat.ResizeOptions) error {
	_, err := c.Client.ExecResize(ctx, id, moby.ExecResizeOptions(options))
	return err
}

func (c *Client) ImageInspectWithRaw(ctx context.Context, id string) (compat.ImageInspect, []byte, error) {
	result, err := c.Client.ImageInspect(ctx, id)
	if err != nil {
		return compat.ImageInspect{}, nil, err
	}
	raw, err := json.Marshal(result.InspectResponse)
	return result.InspectResponse, raw, err
}

func (c *Client) ImageList(ctx context.Context, options compat.ImageListOptions) ([]image.Summary, error) {
	result, err := c.Client.ImageList(ctx, options)
	return result.Items, err
}

func (c *Client) ImagePull(ctx context.Context, ref string, options compat.ImagePullOptions) (io.ReadCloser, error) {
	return c.Client.ImagePull(ctx, ref, options)
}

func (c *Client) ImageRemove(ctx context.Context, id string, options compat.ImageRemoveOptions) ([]image.DeleteResponse, error) {
	result, err := c.Client.ImageRemove(ctx, id, options)
	return result.Items, err
}

func (c *Client) ImageBuild(ctx context.Context, buildContext io.Reader, options compat.ImageBuildOptions) (moby.ImageBuildResult, error) {
	return c.Client.ImageBuild(ctx, buildContext, options)
}

func (c *Client) ImageTag(ctx context.Context, source, target string) error {
	_, err := c.Client.ImageTag(ctx, moby.ImageTagOptions{Source: source, Target: target})
	return err
}

func (c *Client) VolumeInspect(ctx context.Context, id string) (volume.Volume, error) {
	result, err := c.Client.VolumeInspect(ctx, id, moby.VolumeInspectOptions{})
	return result.Volume, err
}

func (c *Client) VolumeList(ctx context.Context, options VolumeListOptions) (moby.VolumeListResult, error) {
	return c.Client.VolumeList(ctx, options)
}

func (c *Client) VolumeCreate(ctx context.Context, options compat.VolumeCreateRequest) (volume.Volume, error) {
	result, err := c.Client.VolumeCreate(ctx, options)
	return result.Volume, err
}

func (c *Client) VolumeRemove(ctx context.Context, id string, force bool) error {
	_, err := c.Client.VolumeRemove(ctx, id, moby.VolumeRemoveOptions{Force: force})
	return err
}

func (c *Client) Info(ctx context.Context) (system.Info, error) {
	result, err := c.Client.Info(ctx, moby.InfoOptions{})
	return result.Info, err
}

func (c *Client) ServerVersion(ctx context.Context) (system.VersionResponse, error) {
	result, err := c.Client.ServerVersion(ctx, moby.ServerVersionOptions{})
	if err != nil {
		return system.VersionResponse{}, err
	}
	version := system.VersionResponse{
		Platform:      system.PlatformInfo{Name: result.Platform.Name},
		Version:       result.Version,
		APIVersion:    result.APIVersion,
		MinAPIVersion: result.MinAPIVersion,
		Os:            result.Os,
		Arch:          result.Arch,
		Experimental:  result.Experimental,
		Components:    result.Components,
	}
	for _, component := range result.Components {
		if component.Name == "Engine" {
			version.GitCommit = component.Details["GitCommit"]
			version.GoVersion = component.Details["GoVersion"]
			version.KernelVersion = component.Details["KernelVersion"]
			version.BuildTime = component.Details["BuildTime"]
			break
		}
	}
	return version, nil
}

func (c *Client) Events(ctx context.Context, options compat.EventsOptions) (<-chan events.Message, <-chan error) {
	result := c.Client.Events(ctx, options)
	return result.Messages, result.Err
}

func IsErrNotFound(err error) bool          { return cerrdefs.IsNotFound(err) }
func IsErrContainerNotFound(err error) bool { return cerrdefs.IsNotFound(err) }
func IsErrImageNotFound(err error) bool     { return cerrdefs.IsNotFound(err) }
func IsErrVolumeNotFound(err error) bool    { return cerrdefs.IsNotFound(err) }
