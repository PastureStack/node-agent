package model

import "github.com/moby/moby/api/types/system"

type InfoData struct {
	Info    system.Info
	Version system.VersionResponse
}
