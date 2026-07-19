package hostinfo

import (
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/pkg/errors"
)

type IopsCollector struct {
}

func (i IopsCollector) GetData() (map[string]interface{}, error) {
	data, err := i.parseIopsData()
	if err != nil {
		return map[string]interface{}{}, errors.Wrap(err, constants.IopsGetDataError+"failed to get data")
	}
	return data, nil
}

func (i IopsCollector) KeyName() string {
	return "iopsInfo"
}

func (i IopsCollector) GetLabels(prefix string) (map[string]string, error) {
	return map[string]string{}, nil
}
