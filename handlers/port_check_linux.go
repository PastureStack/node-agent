package handlers

import (
	"bufio"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func hostSocketConflicts(requested []portCheckPort, dockerBindings map[string]bool) ([]portCheckConflict, bool) {
	root := os.Getenv("CATTLE_HOST_PROC")
	if root == "" {
		root = "/host/proc"
	}
	if info, err := os.Stat(filepath.Join(root, "net")); err != nil || !info.IsDir() {
		return nil, false
	}

	conflicts := []portCheckConflict{}
	complete := true
	files := []struct {
		name     string
		protocol string
		ipv6     bool
	}{
		{"tcp", "tcp", false},
		{"tcp6", "tcp", true},
		{"udp", "udp", false},
		{"udp6", "udp", true},
	}
	for _, spec := range files {
		values, err := parseProcNet(filepath.Join(root, "net", spec.name), spec.protocol, spec.ipv6)
		if err != nil {
			complete = false
			continue
		}
		for _, value := range values {
			if !matchesRequested(requested, value.BindAddress, value.PublicPort, value.Protocol) ||
				explainedByDocker(dockerBindings, value.BindAddress, value.PublicPort, value.Protocol) {
				continue
			}
			conflicts = append(conflicts, value)
		}
	}
	return conflicts, complete
}

func parseProcNet(path, protocol string, ipv6 bool) ([]portCheckConflict, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	result := []portCheckConflict{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 || !strings.Contains(fields[1], ":") {
			continue
		}
		state := strings.ToUpper(fields[3])
		if (protocol == "tcp" && state != "0A") || (protocol == "udp" && state != "07") {
			continue
		}
		parts := strings.SplitN(fields[1], ":", 2)
		portValue, err := strconv.ParseUint(parts[1], 16, 16)
		if err != nil || portValue == 0 {
			continue
		}
		address, err := decodeProcAddress(parts[0], ipv6)
		if err != nil {
			continue
		}
		result = append(result, portCheckConflict{
			Source:      "hostProcess",
			State:       "running",
			BindAddress: address,
			PublicPort:  int(portValue),
			Protocol:    protocol,
		})
	}
	return result, scanner.Err()
}

func decodeProcAddress(value string, ipv6 bool) (string, error) {
	bytes, err := hex.DecodeString(value)
	if err != nil {
		return "", err
	}
	if !ipv6 && len(bytes) == 4 {
		for left, right := 0, len(bytes)-1; left < right; left, right = left+1, right-1 {
			bytes[left], bytes[right] = bytes[right], bytes[left]
		}
	} else if ipv6 && len(bytes) == 16 {
		for offset := 0; offset < 16; offset += 4 {
			bytes[offset], bytes[offset+3] = bytes[offset+3], bytes[offset]
			bytes[offset+1], bytes[offset+2] = bytes[offset+2], bytes[offset+1]
		}
	}
	return normalizeBindAddress(net.IP(bytes).String()), nil
}
