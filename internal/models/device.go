package models

import "time"

// Device represents a discovered host/switch/router/etc.
type Device struct {
	ID          string    `db:"id" json:"id"`
	IPv4        string    `db:"ipv4" json:"ipv4"`
	IPv6        string    `db:"ipv6" json:"ipv6"`
	MAC         string    `db:"mac" json:"mac"`
	Vendor      string    `db:"vendor" json:"vendor"`
	Type        string    `db:"type" json:"type"`
	Hostname    string    `db:"hostname" json:"hostname"`
	SysObjectID string    `db:"sys_object_id" json:"sys_object_id"`
	LastSeen    time.Time `db:"last_seen" json:"last_seen"`
	Confidence  float64   `db:"confidence" json:"confidence"`
}
