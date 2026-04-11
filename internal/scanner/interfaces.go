package scanner

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DetectedCIDR represents a network target found on the host.
type DetectedCIDR struct {
	CIDR      *net.IPNet
	Interface string
}

// DetectDefaultCIDRs inspects the system routing table to locate the default
// interface/CIDR. If parsing fails, it falls back to the first active
// non-loopback interface.
func DetectDefaultCIDRs() ([]DetectedCIDR, error) {
	ifaceName, err := defaultRouteIface("/proc/net/route")
	if err != nil || ifaceName == "" {
		ifaceName = firstCandidateIface()
	}
	if ifaceName == "" {
		return nil, errors.New("no active interface found")
	}
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, err
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}

	var detected []DetectedCIDR
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok || ipnet.IP == nil || ipnet.IP.To4() == nil {
			continue
		}
		ipnet = canonicalize(ipnet)
		detected = append(detected, DetectedCIDR{CIDR: ipnet, Interface: iface.Name})
	}
	if len(detected) == 0 {
		return nil, fmt.Errorf("interface %s has no IPv4 addresses", iface.Name)
	}
	return dedupeCIDRs(detected), nil
}

// LocalInterfaces returns network interfaces suitable for scanning.
func LocalInterfaces() ([]*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var filtered []*net.Interface
	for i := range ifaces {
		iface := ifaces[i]
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

func defaultRouteIface(path string) (string, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Skip header
	if !scanner.Scan() {
		return "", errors.New("empty route table")
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 11 {
			continue
		}
		dest := fields[1]
		flagsHex := fields[3]
		if dest != "00000000" {
			continue
		}
		flags, err := strconv.ParseInt(flagsHex, 16, 32)
		if err != nil {
			continue
		}
		// Flag 0x2 indicates route is up
		if flags&0x2 == 0 {
			continue
		}
		return fields[0], nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("default route not found")
}

func firstCandidateIface() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		return iface.Name
	}
	return ""
}

func canonicalize(ipnet *net.IPNet) *net.IPNet {
	maskOnes, bits := ipnet.Mask.Size()
	if bits != 32 {
		return ipnet
	}
	if maskOnes == 32 {
		maskOnes = 24
	}
	mask := net.CIDRMask(maskOnes, 32)
	ip := ipnet.IP.Mask(mask)
	ipCopy := make(net.IP, len(ip))
	copy(ipCopy, ip)
	return &net.IPNet{IP: ipCopy, Mask: mask}
}

func dedupeCIDRs(cidrs []DetectedCIDR) []DetectedCIDR {
	seen := make(map[string]bool)
	var out []DetectedCIDR
	for _, c := range cidrs {
		key := fmt.Sprintf("%s-%s", c.Interface, c.CIDR.String())
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, c)
	}
	return out
}
