package scanner

import (
	"net"
)

// Job describes a scanning unit.
type Job struct {
	CIDR      *net.IPNet
	Interface string
	ScanID    string
}

// Runner executes a scanning protocol.
type Runner interface {
	Name() string
	Run(job Job, recorder Recorder) error
}
