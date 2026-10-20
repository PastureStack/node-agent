//go:build linux

package hostinfo

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// gopsutil enumerates /proc/1's mounts by default, even when the agent has
// another mount namespace. Resolve usage through that same process's root.
func diskUsageRoot() string {
	proc := os.Getenv("HOST_PROC")
	if proc == "" {
		proc = "/proc"
	}
	process := filepath.Join(proc, "1")
	if override := os.Getenv("HOST_PROC_MOUNTINFO"); override != "" {
		process = filepath.Dir(override)
	} else if !hasDiskMountTable(process) {
		// Match gopsutil's fallback when PID 1's mount table is unavailable.
		process = filepath.Join(proc, "self")
	}

	hostNamespace, hostErr := diskMountNamespace(process)
	selfNamespace, selfErr := diskMountNamespace(filepath.Join(proc, "self"))
	if hostErr == nil && selfErr == nil && hostNamespace == selfNamespace {
		// Preserve direct host operation without requiring access to PID 1's root.
		return ""
	}
	return filepath.Join(process, "root")
}

// The configured proc directory is operator-owned, like gopsutil's mount-table
// source. Read only the fixed namespace link within it; its target is an
// identifier, never a filesystem path to follow (proc namespace links are magic).
func diskMountNamespace(process string) (uint64, error) {
	directory, err := os.OpenRoot(process)
	if err != nil {
		return 0, err
	}
	defer directory.Close()
	identifier, err := directory.Readlink("ns/mnt")
	if err != nil {
		return 0, err
	}
	if !strings.HasPrefix(identifier, "mnt:[") || !strings.HasSuffix(identifier, "]") {
		return 0, os.ErrInvalid
	}
	value := identifier[len("mnt:[") : len(identifier)-1]
	for _, digit := range value {
		if digit < '0' || digit > '9' {
			return 0, os.ErrInvalid
		}
	}
	namespace, err := strconv.ParseUint(value, 10, 64)
	if err != nil || namespace == 0 {
		return 0, os.ErrInvalid
	}
	return namespace, nil
}

func hasDiskMountTable(process string) bool {
	return hasDiskMountTableWithOpen(process, os.Open)
}

func hasDiskMountTableWithOpen(process string, openFile func(string) (*os.File, error)) bool {
	file, err := openFile(filepath.Join(process, "mountinfo"))
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		// Match gopsutil for every open path error, including permission errors.
		file, err = openFile(filepath.Join(process, "mounts"))
	}
	if err != nil {
		return false
	}
	file.Close()
	return true
}
