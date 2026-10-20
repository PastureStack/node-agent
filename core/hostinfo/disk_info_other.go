//go:build !linux

package hostinfo

func diskUsageRoot() string {
	return ""
}
