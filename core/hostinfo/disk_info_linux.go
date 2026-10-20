//go:build linux

package hostinfo

import (
	"errors"
	"os"
	"path/filepath"
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

	hostNamespace, hostErr := os.Stat(filepath.Join(process, "ns", "mnt"))
	selfNamespace, selfErr := os.Stat(filepath.Join(proc, "self", "ns", "mnt"))
	if hostErr == nil && selfErr == nil && os.SameFile(hostNamespace, selfNamespace) {
		// Preserve direct host operation without requiring access to PID 1's root.
		return ""
	}
	return filepath.Join(process, "root")
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
