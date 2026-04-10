package scanner

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// ICMPRunner sends ICMP echo (simple placeholder).
type ICMPRunner struct {
	timeout time.Duration
	log     Logger
}

func NewICMPRunner(log Logger) *ICMPRunner {
	return &ICMPRunner{timeout: 500 * time.Millisecond, log: log}
}

func (r *ICMPRunner) Name() string { return "icmp" }

func (r *ICMPRunner) Run(job Job) error {
	if job.CIDR == nil {
		return fmt.Errorf("icmp: missing CIDR")
	}
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return err
	}
	defer conn.Close()
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: 1, Data: []byte("lanmapper")},
	}
	b, err := msg.Marshal(nil)
	if err != nil {
		return err
	}
	for _, ip := range hostsFromCIDR(job.CIDR) {
		if ip.To4() == nil {
			continue
		}
		if _, err := conn.WriteTo(b, &net.IPAddr{IP: ip}); err != nil {
			r.log.Error("icmp send", "ip", ip.String(), "err", err)
			continue
		}
		_ = conn.SetReadDeadline(time.Now().Add(r.timeout))
		buf := make([]byte, 1500)
		if _, _, err := conn.ReadFrom(buf); err == nil {
			r.log.Info("icmp reply", "ip", ip.String(), "scan", job.ScanID)
		}
	}
	return nil
}
