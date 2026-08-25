package compute

import (
	"github.com/PastureStack/node-agent/internal/dockerapi/client"
	"github.com/PastureStack/node-agent/model"
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/PastureStack/node-agent/utilities/utils"
	"github.com/pkg/errors"
)

func IsInstanceActive(instance model.Instance, host model.Host, client *client.Client) (bool, error) {
	if utils.IsNoOp(instance.ProcessData) {
		return true, nil
	}

	container, err := utils.GetContainer(client, instance, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return false, nil
		}
		return false, errors.Wrap(err, constants.IsInstanceActiveError+"failed to get container")
	}
	return isRunning(client, container)
}

func IsInstanceInactive(instance model.Instance, client *client.Client) (bool, error) {
	if utils.IsNoOp(instance.ProcessData) {
		return true, nil
	}

	container, err := utils.GetContainer(client, instance, false)
	if err != nil {
		if !utils.IsContainerNotFoundError(err) {
			return false, errors.Wrap(err, constants.IsInstanceInactiveError+"failed to get container")
		}
	}
	return isStopped(client, container)
}

func IsInstanceRemoved(instance model.Instance, dockerClient *client.Client) (bool, error) {
	con, err := utils.GetContainer(dockerClient, instance, false)
	if err != nil {
		if utils.IsContainerNotFoundError(err) {
			return true, nil
		}
		return false, errors.Wrap(err, constants.IsInstanceRemovedError+"failed to get container")
	}
	return con.ID == "", nil
}
