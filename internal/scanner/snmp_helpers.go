package scanner

import (
	"fmt"
	"net"
	"strings"

	"github.com/gosnmp/gosnmp"
)

type snmpClient interface {
	Get(oids []string) (*gosnmp.SnmpPacket, error)
	Walk(oid string, walkFn gosnmp.WalkFunc) error
	Close()
}

type snmpFactory func(target, community string) (snmpClient, error)

type goSNMPClient struct {
	inner *gosnmp.GoSNMP
}

func (c *goSNMPClient) Get(oids []string) (*gosnmp.SnmpPacket, error) {
	return c.inner.Get(oids)
}

func (c *goSNMPClient) Walk(oid string, walkFn gosnmp.WalkFunc) error {
	return c.inner.Walk(oid, walkFn)
}

func (c *goSNMPClient) Close() {
	c.inner.Conn.Close()
}

func defaultSNMPFactory(target, community string) (snmpClient, error) {
	client := &gosnmp.GoSNMP{
		Target:    target,
		Community: community,
		Port:      161,
		Version:   gosnmp.Version2c,
		Timeout:   defaultSNMPTimeout,
		Retries:   1,
	}
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("snmp connect %s: %w", target, err)
	}
	return &goSNMPClient{inner: client}, nil
}

func sanitizeName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.ReplaceAll(trimmed, " ", "_")
	trimmed = strings.ToLower(trimmed)
	return trimmed
}

func macFromBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	hw := net.HardwareAddr(b)
	return strings.ToLower(hw.String())
}
