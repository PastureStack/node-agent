package hostinfo

import (
	"errors"
	"math"
	"path/filepath"
	"strings"

	"github.com/PastureStack/node-agent/model"
	"github.com/PastureStack/node-agent/utilities/utils"
	"github.com/shirou/gopsutil/v4/disk"
)

func (d DiskCollector) convertUnits(number uint64) float64 {
	// return in MB
	return math.Floor(float64(number/d.Unit*1000)) / 1000.0
}

func (d DiskCollector) getDockerStorageInfo(infoData model.InfoData) map[string]interface{} {
	data := map[string]interface{}{}

	info := infoData.Info
	for _, item := range info.DriverStatus {
		data[item[0]] = item[1]
	}
	return data
}

func (d DiskCollector) includeInFilesystem(infoData model.InfoData, device string) bool {
	include := true
	if infoData.Info.Driver == "devicemapper" {
		pool := d.getDockerStorageInfo(infoData)
		poolName, ok := pool["Pool Name"]
		if !ok {
			poolName = "/dev/mapper/docker-"
		}
		if strings.HasSuffix(utils.InterfaceToString(poolName), "-pool") {
			poolName := utils.InterfaceToString(poolName)
			poolName = poolName[len(poolName)-5 : len(poolName)]
		}
		if strings.Contains(device, utils.InterfaceToString(poolName)) {
			include = false
		}
	}
	return include
}

func (d DiskCollector) collectDiskInfo(infoData model.InfoData) (map[string]interface{}, map[string]interface{}, error) {
	fileSystems := map[string]interface{}{}
	mountPoints := map[string]interface{}{}
	partitions, err := disk.Partitions(false)
	if err != nil {
		return fileSystems, mountPoints, err
	}
	root := diskUsageRoot()
	var usageErrors []error
	for _, partition := range partitions {
		// Snap packages are read-only filesystem images, not host disk capacity.
		if partition.Fstype == "squashfs" {
			continue
		}
		mountpoint := partition.Mountpoint
		if root != "" {
			mountpoint = filepath.Join(root, mountpoint)
		}
		usage, err := disk.Usage(mountpoint)
		if err != nil {
			usageErrors = append(usageErrors, err)
			continue
		}
		mountPoints[partition.Device] = map[string]interface{}{
			"free":       d.convertUnits(usage.Free),
			"total":      d.convertUnits(usage.Total),
			"used":       d.convertUnits(usage.Used),
			"percentage": usage.UsedPercent,
		}
		if d.includeInFilesystem(infoData, partition.Device) {
			fileSystems[partition.Device] = map[string]interface{}{
				"capacity": d.convertUnits(usage.Total),
			}
		}
	}
	return fileSystems, mountPoints, errors.Join(usageErrors...)
}
