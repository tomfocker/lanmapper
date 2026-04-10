package scanner

import (
	"context"
	"fmt"
	"net"

	"golang.org/x/time/rate"
)

// ARPNDRunner emits ARP/ND probes (currently placeholder logging only).
type ARPNDRunner struct {
	limiter *rate.Limiter
	log     Logger
}

// NewARPNDRunner builds runner with default rate limit.
func NewARPNDRunner(log Logger) *ARPNDRunner {
	return &ARPNDRunner{
		limiter: rate.NewLimiter(rate.Limit(200), 400),
		log:     log,
	}
}

func (r *ARPNDRunner) Name() string { return "arp_nd" }

func (r *ARPNDRunner) Run(job Job) error {
	if job.CIDR == nil {
		return fmt.Errorf("arp_nd: missing CIDR")
	}
	ctx := context.Background()
	for _, ip := range hostsFromCIDR(job.CIDR) {
		if err := r.limiter.Wait(ctx); err != nil {
			return err
		}
		r.log.Info("arp_nd probe", "ip", ip.String(), "iface", job.Interface, "scan", job.ScanID)
	}
	return nil
}

func hostsFromCIDR(cidr *net.IPNet) []net.IP {
	var hosts []net.IP
	start := append(net.IP(nil), cidr.IP.Mask(cidr.Mask)...)
	end := broadcastAddr(cidr)
	maxSamples := 1 << 16 // prevent infinite on IPv6
	for ip, count := start, 0; cidr.Contains(ip); ip, count = incIP(ip), count+1 {
		if ip.Equal(start) && start.To4() != nil {
			continue
		}
		if end != nil && ip.Equal(end) {
			continue
		}
		hosts = append(hosts, append(net.IP(nil), ip...))
		if end != nil && ip.Equal(end) {
			break
		}
		if end == nil && count > maxSamples {
			break
		}
	}
	return hosts
}

func incIP(ip net.IP) net.IP {
	out := append(net.IP(nil), ip...)
	for i := len(out) - 1; i >= 0; i-- {
		out[i]++
		if out[i] != 0 {
			break
		}
	}
	return out
}

func broadcastAddr(cidr *net.IPNet) net.IP {
	ip := cidr.IP.To4()
	if ip == nil {
		return nil
	}
	mask := cidr.Mask
	out := make(net.IP, len(ip))
	copy(out, ip)
	for i := range out {
		out[i] |= ^mask[i]
	}
	return out
}
