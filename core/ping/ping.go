package ping

import (
	"github.com/PastureStack/node-agent/core/hostinfo"
	"github.com/PastureStack/node-agent/model"
	"github.com/PastureStack/node-agent/utilities/config"
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	revents "github.com/rancher/event-subscriber/events"
)

func DoPingAction(event *revents.Event, resp *model.PingResponse, dockerClient *client.Client, collectors []hostinfo.Collector) error {
	if !config.DockerEnable() {
		return nil
	}
	if err := addResource(event, resp, dockerClient, collectors); err != nil {
		return errors.Wrap(err, constants.DoPingActionError+"failed to add resource")
	}
	if err := addInstance(event, resp, dockerClient); err != nil {
		return errors.Wrap(err, constants.DoPingActionError+"failed to add instance")
	}
	return nil
}
