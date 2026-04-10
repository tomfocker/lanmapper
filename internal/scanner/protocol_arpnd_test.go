package scanner

import (
	"net"
	"testing"
)

func TestHostsFromCIDR(t *testing.T) {
	_, cidr, _ := net.ParseCIDR("192.168.1.0/30")
	hosts := hostsFromCIDR(cidr)
	if len(hosts) != 2 || hosts[0].String() != "192.168.1.1" || hosts[1].String() != "192.168.1.2" {
		t.Fatalf("unexpected hosts: %+v", hosts)
	}
}
