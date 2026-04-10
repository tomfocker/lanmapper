package models

// Interface captures properties of a device port.
type Interface struct {
	ID        string `db:"id" json:"id"`
	DeviceID  string `db:"device_id" json:"device_id"`
	Name      string `db:"name" json:"name"`
	MAC       string `db:"mac" json:"mac"`
	SpeedMbps int64  `db:"speed_mbps" json:"speed_mbps"`
	VLAN      string `db:"vlan" json:"vlan"`
	IsUplink  bool   `db:"is_uplink" json:"is_uplink"`
}
