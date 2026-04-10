package scanner

import (
	"net"
)

// LocalInterfaces returns network interfaces suitable for scanning.
func LocalInterfaces() ([]*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var filtered []*net.Interface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		filtered = append(filtered, &iface)
	}
	return filtered, nil
}
