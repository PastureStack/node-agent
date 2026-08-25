package events

import (
	"context"
	"github.com/PastureStack/node-agent/internal/dockerapi/types"
)

type SimpleDockerClient interface {
	ContainerInspect(context context.Context, id string) (types.ContainerJSON, error)
}
