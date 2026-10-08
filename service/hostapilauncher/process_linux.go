//go:build linux

package hostapilauncher

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func bindParentLifetime(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
}

func validateProducerOwner(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || (stat.Uid != 0 && stat.Uid != uint32(os.Geteuid())) {
		return errors.New("producer owner is not trusted")
	}
	return nil
}
