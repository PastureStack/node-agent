package handlers

import (
	"github.com/PastureStack/node-agent/core/hostinfo"
	"github.com/PastureStack/node-agent/core/marshaller"
	"github.com/PastureStack/node-agent/core/ping"
	engineCli "github.com/PastureStack/node-agent/internal/dockerapi/client"
	"github.com/PastureStack/node-agent/model"
	"github.com/PastureStack/node-agent/utilities/config"
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/pkg/errors"
	revents "github.com/rancher/event-subscriber/events"
	"github.com/rancher/go-rancher/v2"
)

type PingHandler struct {
	dockerClient *engineCli.Client
	collectors   []hostinfo.Collector
}

func (h *PingHandler) Ping(event *revents.Event, cli *client.RancherClient) error {
	if event.ReplyTo == "" {
		return nil
	}
	resp := model.PingResponse{
		Resources: []model.PingResource{},
	}
	if config.DoPing() {
		if err := ping.DoPingAction(event, &resp, h.dockerClient, h.collectors); err != nil {
			return errors.Wrap(err, constants.PingError+"failed to do ping action")
		}
	}
	data, err := marshaller.StructToMap(resp)
	if err != nil {
		return errors.Wrap(err, constants.PingError+"failed to marshall response data")
	}
	return reply(data, event, cli)
}
