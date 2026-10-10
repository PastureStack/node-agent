//go:build !linux

package hostapilauncher

import (
	"os"
	"os/exec"
)

func bindParentLifetime(*exec.Cmd)            {}
func validateProducerOwner(os.FileInfo) error { return nil }
