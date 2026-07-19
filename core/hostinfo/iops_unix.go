//go:build linux || freebsd || solaris || openbsd || darwin
// +build linux freebsd solaris openbsd darwin

package hostinfo

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/PastureStack/node-agent/utilities/utils"
	"github.com/pkg/errors"
)

func (i IopsCollector) getIopsData(readOrWrite string) (map[string]interface{}, error) {
	file, err := os.Open("/var/lib/rancher/state/" + readOrWrite + ".json")
	if err != nil {
		return map[string]interface{}{}, err
	}
	defer file.Close()
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return map[string]interface{}{}, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		return map[string]interface{}{}, err
	}
	return result, nil
}

func (i IopsCollector) parseIopsData() (map[string]interface{}, error) {
	data := map[string]interface{}{}
	readJSONData, err := i.getIopsData("read")
	if err != nil && !os.IsNotExist(err) {
		return data, errors.Wrap(err, constants.ParseIopsDataError+"failed to read iops file")
	} else if err != nil && os.IsNotExist(err) {
		return data, nil
	}
	writeJSONData, err := i.getIopsData("write")
	if err != nil && !os.IsNotExist(err) {
		return data, errors.Wrap(err, constants.ParseIopsDataError+"failed to read iops file")
	} else if err != nil && os.IsNotExist(err) {
		return data, nil
	}
	readJob, ok := firstIopsEntry(readJSONData, "jobs")
	if !ok {
		return data, errors.New(constants.ParseIopsDataError + "missing read job data")
	}
	writeJob, ok := firstIopsEntry(writeJSONData, "jobs")
	if !ok {
		return data, errors.New(constants.ParseIopsDataError + "missing write job data")
	}
	diskUtil, ok := firstIopsEntry(readJSONData, "disk_util")
	if !ok {
		return data, errors.New(constants.ParseIopsDataError + "missing disk utilization data")
	}
	readIops, _ := utils.GetFieldsIfExist(readJob, "read", "iops")
	writeIops, _ := utils.GetFieldsIfExist(writeJob, "write", "iops")
	device, ok := utils.GetFieldsIfExist(diskUtil, "name")
	if !ok {
		return data, errors.New(constants.ParseIopsDataError + "missing disk name")
	}
	deviceName, ok := device.(string)
	if !ok {
		return data, errors.New(constants.ParseIopsDataError + "disk name is not a string")
	}
	key := "/dev/" + deviceName
	data[key] = map[string]interface{}{
		"read":  readIops,
		"write": writeIops,
	}
	return data, nil
}

func firstIopsEntry(data map[string]interface{}, field string) (map[string]interface{}, bool) {
	rawEntries, ok := data[field].([]interface{})
	if !ok || len(rawEntries) == 0 {
		return nil, false
	}
	entry, ok := rawEntries[0].(map[string]interface{})
	return entry, ok
}

func (i IopsCollector) getDefaultDisk() (string, error) {
	data, err := i.GetData()
	if err != nil {
		return "", errors.Wrap(err, constants.GetDefaultDiskError+"failed to get data")
	}
	if len(data) == 0 {
		return "", nil
	}
	for key := range data {
		return key, nil
	}
	return "", nil
}
