package scanner

import (
	"time"

	"github.com/gosnmp/gosnmp"
)

// SNMPRunner polls devices for LLDP/CDP/Bridge data.
type SNMPRunner struct {
	communities []string
	timeout     time.Duration
	log         Logger
}

func NewSNMPRunner(log Logger, communities []string) *SNMPRunner {
	if len(communities) == 0 {
		communities = []string{"public", "private"}
	}
	return &SNMPRunner{communities: communities, timeout: 2 * time.Second, log: log}
}

func (r *SNMPRunner) Name() string { return "snmp" }

func (r *SNMPRunner) Run(job Job, recorder Recorder) error {
	ips := hostsFromCIDR(job.CIDR)
	for _, ip := range ips {
		for _, community := range r.communities {
			g := &gosnmp.GoSNMP{
				Target:    ip.String(),
				Community: community,
				Port:      161,
				Version:   gosnmp.Version2c,
				Timeout:   r.timeout,
				Retries:   1,
			}
			if err := g.Connect(); err != nil {
				r.log.Error("snmp connect", "ip", ip.String(), "err", err)
				continue
			}
			_, err := g.Get([]string{"1.3.6.1.2.1.1.5.0"}) // sysName
			g.Conn.Close()
			if err == nil {
				r.log.Info("snmp reachable", "ip", ip.String(), "community", community)
				break
			}
		}
	}
	return nil
}
