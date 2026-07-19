//go:build linux || freebsd || solaris || openbsd
// +build linux freebsd solaris openbsd

package constants

import "syscall"

var SysAttr = &syscall.SysProcAttr{
	Setpgid:   true,
	Pdeathsig: syscall.SIGKILL,
}
