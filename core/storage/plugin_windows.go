package storage

import (
	"github.com/PastureStack/node-agent/model"
)

// Platform storage volume attachment is not supported on Windows.
func CallPlatformStorageVolumePlugin(volume model.Volume, action string, payload interface{}) (Response, error) {
	return Response{}, nil
}
