package events

import (
	"context"

	"github.com/PastureStack/node-agent/internal/dockerapi/client"
	"github.com/PastureStack/node-agent/service/hostapi/config"
	"github.com/PastureStack/node-agent/service/hostapi/util"
	"github.com/moby/moby/api/types/events"
	rclient "github.com/rancher/go-rancher/client"
)

const (
	simulatedEvent = "-simulated-"
)

func NewDockerEventsProcessor(poolSize int) *DockerEventsProcessor {
	return &DockerEventsProcessor{
		poolSize:          poolSize,
		getDockerClient:   getDockerClientFn,
		getHandlers:       getHandlersFn,
		getPlatformClient: util.GetPlatformClient,
	}
}

type DockerEventsProcessor struct {
	poolSize          int
	getDockerClient   func() (*client.Client, error)
	getHandlers       func(*client.Client, *rclient.RancherClient) (map[string][]Handler, error)
	getPlatformClient func() (*rclient.RancherClient, error)
}

func (de *DockerEventsProcessor) Process() error {
	dockerClient, err := de.getDockerClient()
	if err != nil {
		return err
	}

	platformClient, err := de.getPlatformClient()
	if err != nil {
		return err
	}

	handlers, err := de.getHandlers(dockerClient, platformClient)
	if err != nil {
		return err
	}

	router, err := NewEventRouter(de.poolSize, de.poolSize, dockerClient, handlers)
	if err != nil {
		return err
	}
	router.Start()

	filter := make(client.Filters)
	filter.Add("status", "paused")
	filter.Add("status", "running")
	listOpts := client.ContainerListOptions{
		All:     true,
		Filters: filter,
	}
	containers, err := dockerClient.ContainerList(context.Background(), listOpts)
	if err != nil {
		return err
	}

	for _, c := range containers {
		event := &events.Message{
			Type:   events.ContainerEventType,
			Action: events.Action("start"),
			Actor: events.Actor{
				ID:         c.ID,
				Attributes: map[string]string{"image": simulatedEvent},
			},
		}
		router.listener <- event
	}
	return nil
}

func getDockerClientFn() (*client.Client, error) {
	return NewDockerClient()
}

func getHandlersFn(dockerClient *client.Client, platformClient *rclient.RancherClient) (map[string][]Handler, error) {

	handlers := map[string][]Handler{}

	// Control-platform event handler.
	if platformClient != nil {
		sendToPlatformHandler := &SendToPlatformHandler{
			client:   dockerClient,
			platform: platformClient,
			hostUUID: getHostUUID(),
		}
		handlers["start"] = append(handlers["start"], sendToPlatformHandler)
		handlers["stop"] = []Handler{sendToPlatformHandler}
		handlers["die"] = []Handler{sendToPlatformHandler}
		handlers["kill"] = []Handler{sendToPlatformHandler}
		handlers["destroy"] = []Handler{sendToPlatformHandler}
	}

	return handlers, nil
}

func getHostUUID() string {
	return config.Config.HostUUID
}
