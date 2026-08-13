//go:build !windows
// +build !windows

package hostinfo

import (
	"bufio"
	"github.com/PastureStack/node-agent/utilities/constants"
	"github.com/pkg/errors"
	"os"
	"regexp"
)

func getKernelVersion() (string, error) {
	file, err := os.Open("/proc/version")
	defer file.Close()
	data := []string{}
	if err != nil {
		return "", errors.Wrap(err, constants.GetKernelVersionError+"failed to open process version file")
	}
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}
	version := regexp.MustCompile("\\d+.\\d+.\\d+").FindString(data[0])
	return version, nil
}

func getOSName() string {
	return "linux"
}
