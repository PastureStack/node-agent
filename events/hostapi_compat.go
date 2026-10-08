//go:build !linux

package events

import (
	"context"

	"github.com/PastureStack/node-agent/service/hostapi"
)

// The existing Windows bootstrap is not a supported production upgrade path.
// Preserve its historical compatibility behavior without claiming new Linux
// delegation or durable completion support on that platform.
func startHostAPI(context.Context) { hostapi.StartUp() }
