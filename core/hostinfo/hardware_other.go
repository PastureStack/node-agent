//go:build !linux

package hostinfo

import "errors"

func discoverHardware(string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, errors.New("Linux GPU discovery is not available on this OS")
}
