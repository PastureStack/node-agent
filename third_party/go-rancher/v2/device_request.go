package client

// DeviceRequest is the LaunchConfig hardware contract, not a scheduler reservation.
type DeviceRequest struct {
	Driver       string            `json:"driver,omitempty" yaml:"driver,omitempty"`
	Count        int               `json:"count,omitempty" yaml:"count,omitempty"`
	DeviceIDs    []string          `json:"deviceIds,omitempty" yaml:"device_ids,omitempty"`
	Capabilities [][]string        `json:"capabilities" yaml:"capabilities"`
	Options      map[string]string `json:"options,omitempty" yaml:"options,omitempty"`
}
