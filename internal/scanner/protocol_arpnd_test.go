package scanner

import (
	"net"
	"os"
	"testing"
)

func TestHostsFromCIDR(t *testing.T) {
	_, cidr, _ := net.ParseCIDR("192.168.1.0/30")
	hosts := hostsFromCIDR(cidr)
	if len(hosts) != 2 || hosts[0].String() != "192.168.1.1" || hosts[1].String() != "192.168.1.2" {
		t.Fatalf("unexpected hosts: %+v", hosts)
	}
}

func TestParseARP(t *testing.T) {
	content := `IP address       HW type     Flags       HW address            Mask     Device
192.168.1.1     0x1         0x2         aa:bb:cc:dd:ee:ff     *        eth0
`
	tmp := t.TempDir()
	path := tmp + "/arp"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	entries, err := parseARP(path)
	if err != nil {
		t.Fatalf("parse arp: %v", err)
	}
	if len(entries) != 1 || entries[0].MAC != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("unexpected entries: %+v", entries)
	}
}
