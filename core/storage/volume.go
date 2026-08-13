package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PastureStack/node-agent/core/progress"
	"github.com/PastureStack/node-agent/model"
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/PastureStack/node-agent/utilities/utils"
	"github.com/docker/docker/api/types"
	engineCli "github.com/docker/docker/client"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
	"github.com/rancher/log"
	"golang.org/x/net/context"
)

const (
	Create               = "Create"
	Remove               = "Remove"
	Attach               = "Attach"
	Mount                = "Mount"
	Path                 = "Path"
	Unmount              = "Unmount"
	Get                  = "Get"
	List                 = "List"
	Capabilities         = "Capabilities"
	legacyStorageSockDir = "/var/run/rancher/storage"
)

// Response is the strucutre that the plugin's responses are serialized to.
type Response struct {
	Mountpoint   string
	Err          string
	Volumes      []*Volume
	Volume       *Volume
	Capabilities Capability
}

// Volume represents a volume object for use with `Get` and `List` requests
type Volume struct {
	Name       string
	Mountpoint string
	Status     map[string]interface{}
}

// Capability represents the list of capabilities a volume driver can return
type Capability struct {
	Scope string
}

var legacyPlatformDrivers = map[string]bool{
	"rancher-ebs":       true,
	"rancher-nfs":       true,
	"rancher-efs":       true,
	"rancher-secrets":   true,
	"secrets-bridge-v2": true,
}

func VolumeActivateDocker(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress, client *engineCli.Client) error {
	if ok, err := IsVolumeActive(volume, storagePool, client); ok {
		return nil
	} else if err != nil {
		return errors.Wrap(err, constants.VolumeActivateError+"failed to check whether volume is activated")
	}

	if err := DoVolumeActivate(volume, storagePool, progress, client); err != nil {
		return errors.Wrap(err, constants.VolumeActivateError+"failed to activate volume")
	}
	if ok, err := IsVolumeActive(volume, storagePool, client); !ok && err != nil {
		return errors.Wrap(err, constants.VolumeActivateError)
	} else if !ok && err == nil {
		return errors.New(constants.VolumeActivateError + "volume is not activated")
	}
	return nil
}

func VolumeRemoveDocker(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress, dockerClient *engineCli.Client, ca *cache.Cache, resourceID string) error {
	if ok, err := IsVolumeRemoved(volume, storagePool, dockerClient); err == nil && !ok {
		rmErr := DoVolumeRemove(volume, storagePool, progress, dockerClient, ca, resourceID)
		if rmErr != nil {
			return errors.Wrap(rmErr, constants.VolumeRemoveError+"failed to remove volume")
		}
	} else if err != nil {
		return errors.Wrap(err, constants.VolumeRemoveError+"failed to check whether volume is removed")
	}
	return nil
}

func VolumeActivateFlex(volume model.Volume) error {
	payload := struct{ Name string }{Name: volume.Name}
	_, err := CallPlatformStorageVolumePlugin(volume, Create, payload)
	if err != nil {
		return err
	}
	return nil
}

func VolumeRemoveFlex(volume model.Volume) error {
	payload := struct{ Name string }{Name: volume.Name}
	_, err := CallPlatformStorageVolumePlugin(volume, Remove, payload)
	if err != nil {
		return err
	}
	return nil
}

func DoVolumeActivate(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress, client *engineCli.Client) error {
	if !isManagedVolume(volume) {
		return nil
	}
	driver := volume.Data.Fields.Driver
	driverOpts := volume.Data.Fields.DriverOpts
	opts := map[string]string{}
	if driverOpts != nil {
		for k, v := range driverOpts {
			opts[k] = utils.InterfaceToString(v)
		}
	}

	// Legacy Longhorn volumes indicate when they have been moved to a
	// different host. If so, we have to delete before we create
	// to cleanup the reference in docker.

	vol, err := client.VolumeInspect(context.Background(), volume.Name)
	if err != nil {
		if vol.Mountpoint == "moved" {
			log.Info(fmt.Sprintf("Removing moved volume %s so that it can be re-added.", volume.Name))
			if err := client.VolumeRemove(context.Background(), volume.Name, true); err != nil {
				return errors.Wrap(err, constants.DoVolumeActivateError+"failed to remove volume")
			}
		}
	}

	options := types.VolumeCreateRequest{
		Name:       volume.Name,
		Driver:     driver,
		DriverOpts: opts,
	}
	_, err1 := client.VolumeCreate(context.Background(), options)
	if err1 != nil {
		return errors.Wrap(err1, constants.DoVolumeActivateError+"failed to create volume")
	}
	return nil
}

func DoVolumeRemove(volume model.Volume, storagePool model.StoragePool, progress *progress.Progress, dockerClient *engineCli.Client, ca *cache.Cache, resourceID string) error {
	if _, ok := ca.Get(resourceID); ok {
		ca.Delete(resourceID)
		return nil
	}
	if ok, err := IsVolumeRemoved(volume, storagePool, dockerClient); ok {
		return nil
	} else if err != nil {
		return errors.Wrap(err, constants.DoVolumeRemoveError+"failed to check whether volume is removed")
	}
	if volume.DeviceNumber == 0 {
		container, err := utils.GetContainer(dockerClient, volume.Instance, false)
		if err != nil {
			if !utils.IsContainerNotFoundError(err) {
				return errors.Wrap(err, constants.DoVolumeRemoveError+"faild to get container")
			}
		}
		if container.ID == "" {
			return nil
		}

		if utils.IsNodeAgentContainer(container) {
			log.Warnf("Received event to delete a root volume for the node-agent container with id [%v]. Dropping event for resource [%v].", container.ID, resourceID)
			return nil
		}

		errorList := []error{}
		for i := 0; i < 3; i++ {
			if err := utils.RemoveContainer(dockerClient, container.ID); err != nil && !engineCli.IsErrContainerNotFound(err) {
				errorList = append(errorList, err)
			} else {
				break
			}
			time.Sleep(time.Second * 1)
		}
		if len(errorList) == 3 {
			ca.Add(resourceID, true, cache.DefaultExpiration)
			log.Warnf("Failed to remove container id [%v]. Tried three times and failed. Error msg: %v", container.ID, errorList)
		}
	} else if isManagedVolume(volume) {
		errorList := []error{}
		for i := 0; i < 3; i++ {
			err := dockerClient.VolumeRemove(context.Background(), volume.Name, false)
			if err != nil {
				if strings.Contains(err.Error(), "Should retry") {
					return errors.Wrap(err, constants.DoVolumeRemoveError+"Error removing volume")
				}
				errorList = append(errorList, err)
			} else {
				break
			}
			time.Sleep(time.Second * 1)
		}
		if len(errorList) == 3 {
			ca.Add(resourceID, true, cache.DefaultExpiration)
			log.Warnf("Failed to remove volume name [%v]. Tried three times and failed. Error msg: %v", volume.Name, errorList)
		}
		return nil
	}
	path := pathToVolume(volume)
	if !volume.Data.Fields.IsHostPath {
		_, existErr := os.Stat(path)
		if existErr == nil {
			if err := os.RemoveAll(path); err != nil {
				return errors.Wrap(err, constants.DoVolumeRemoveError+"failed to remove directory")
			}
		}
	}
	return nil
}

func isManagedVolume(volume model.Volume) bool {
	driver := volume.Data.Fields.Driver
	if driver == "" {
		return false
	}
	if _, ok := legacyPlatformDrivers[driver]; ok {
		return true
	}
	if volume.Name == "" {
		return false
	}
	return true
}

func pathToVolume(volume model.Volume) string {
	return strings.Replace(volume.URI, "file://", "", -1)
}

func IsVolumeActive(volume model.Volume, storagePool model.StoragePool, dockerClient *engineCli.Client) (bool, error) {
	if !isManagedVolume(volume) {
		return true, nil
	}
	vol, err := dockerClient.VolumeInspect(context.Background(), volume.Name)
	if engineCli.IsErrVolumeNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, errors.Wrap(err, constants.IsVolumeActiveError)
	}
	if vol.Mountpoint != "" {
		return vol.Mountpoint != "moved", nil
	}
	return true, nil
}

func platformStorageSockPath(volume model.Volume) string {
	return filepath.Join(legacyStorageSockDir, volume.Data.Fields.Driver+".sock")
}

// IsPlatformVolume checks whether a legacy platform driver exposes flex-volume capabilities.
// It returns an error when the managed driver is configured but its socket is unavailable.
func IsPlatformVolume(volume model.Volume) (bool, error) {
	if _, ok := legacyPlatformDrivers[volume.Data.Fields.Driver]; ok {
		if _, err := os.Stat(platformStorageSockPath(volume)); err == nil {
			// check if Capabilities is flex
			payload := struct {
				Name    string
				Options map[string]string `json:"Opts,omitempty"`
			}{
				Name:    volume.Name,
				Options: volume.Data.Fields.DriverOpts,
			}
			response, err := CallPlatformStorageVolumePlugin(volume, Capabilities, payload)
			if err != nil {
				return false, err
			}
			if response.Capabilities.Scope == "flex" {
				return true, nil
			}
			return false, nil
		}
		return false, errors.Errorf("socket file not found at %s", platformStorageSockPath(volume))
	}
	return false, nil
}

// IsPlatformManagedDriver checks whether a volume uses a managed legacy driver.
func IsPlatformManagedDriver(volume model.Volume) (bool, error) {
	if _, ok := legacyPlatformDrivers[volume.Data.Fields.Driver]; ok {
		if _, err := os.Stat(platformStorageSockPath(volume)); err == nil {
			return true, nil
		}
		return false, errors.Errorf("legacy platform driver %s is not running: can't find socket file", volume.Driver)
	}
	return false, nil
}

func IsVolumeRemoved(volume model.Volume, storagePool model.StoragePool, client *engineCli.Client) (bool, error) {
	if volume.DeviceNumber == 0 {
		container, err := utils.GetContainer(client, volume.Instance, false)
		if err != nil {
			if !utils.IsContainerNotFoundError(err) {
				return false, errors.Wrap(err, constants.IsVolumeRemovedError+"failed to get container")
			}
		}
		return container.ID == "", nil
	} else if isManagedVolume(volume) {
		ok, err := IsVolumeActive(volume, storagePool, client)
		if err != nil {
			return false, errors.Wrap(err, constants.IsVolumeRemovedError+"failed to check whether volume is activated")
		}
		return !ok, nil
	}
	path := pathToVolume(volume)
	if !volume.Data.Fields.IsHostPath {
		return true, nil
	}
	_, exist := os.Stat(path)
	if exist != nil {
		return true, nil
	}
	return false, nil
}
