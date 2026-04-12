package models

// Link represents a connection between two interfaces.
type Link struct {
	ID         string  `db:"id" json:"id"`
	ADevice    string  `db:"a_device" json:"a_device"`
	AInterface string  `db:"a_interface" json:"a_interface"`
	BDevice    string  `db:"b_device" json:"b_device"`
	BInterface string  `db:"b_interface" json:"b_interface"`
	Media      string  `db:"media" json:"media"`
	SpeedMbps  int64   `db:"speed_mbps" json:"speed_mbps"`
	Source     string  `db:"source" json:"source"`
	Kind       string  `db:"kind" json:"kind"`
	Confidence float64 `db:"confidence" json:"confidence"`
}
