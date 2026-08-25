package events

import (
	"context"
	"github.com/PastureStack/node-agent/internal/dockerapi/client"
	"github.com/moby/moby/api/types/events"
	"github.com/rancher/event-subscriber/locks"
	rclient "github.com/rancher/go-rancher/client"
	"github.com/rancher/log"
)

type SendToPlatformHandler struct {
	client   *client.Client
	platform *rclient.RancherClient
	hostUUID string
}

func (h *SendToPlatformHandler) Handle(event *events.Message) error {
	status := string(event.Action)
	id := event.Actor.ID
	from := ""
	if event.Actor.Attributes != nil {
		from = event.Actor.Attributes["image"]
	}
	// The compatibility state watcher sends a simulated event to initiate IP injection.
	// This event should not be sent.
	if from == simulatedEvent {
		return nil
	}

	// Note: event.ID == container's ID
	lock := locks.Lock(status + id)
	if lock == nil {
		log.Debugf("Container locked. Can't run SendToPlatformHandler. Event: [%s], ID: [%s]", status, id)
		return nil
	}
	defer lock.Unlock()

	container, err := h.client.ContainerInspect(context.Background(), id)
	if err != nil {
		if ok := client.IsErrContainerNotFound(err); !ok {
			return err
		}
	}

	containerEvent := &rclient.ContainerEvent{
		ExternalStatus:    status,
		ExternalId:        id,
		ExternalFrom:      from,
		ExternalTimestamp: int64(event.Time),
		ReportedHostUuid:  h.hostUUID,
	}
	containerEvent.DockerInspect = container

	if _, err := h.platform.ContainerEvent.Create(containerEvent); err != nil {
		return err
	}

	return nil
}
