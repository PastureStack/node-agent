package hostinfo

import (
	"github.com/PastureStack/node-agent/model"
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/pkg/errors"
)

type DiskCollector struct {
	Unit                uint64
	InfoData            model.InfoData
	DockerStorageDriver string
}

func (d DiskCollector) GetData() (map[string]interface{}, error) {
	infoData := d.InfoData
	data := map[string]interface{}{
		"fileSystems":               map[string]interface{}{},
		"mountPoints":               map[string]interface{}{},
		"dockerStorageDriverStatus": map[string]interface{}{},
		"dockerStorageDriver":       infoData.Info.Driver,
	}

	fileSystems, mountPoints, err := d.collectDiskInfo(infoData)
	data["fileSystems"] = fileSystems
	data["mountPoints"] = mountPoints
	if err != nil {
		return data, errors.Wrap(err, constants.DiskGetDataError+"failed get filesystem info")
	}

	return data, nil
}

func (d DiskCollector) KeyName() string {
	return "diskInfo"
}

func (d DiskCollector) GetLabels(prefix string) (map[string]string, error) {
	return map[string]string{}, nil
}
