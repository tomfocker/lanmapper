package scanner

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

const (
	oidSysName         = "1.3.6.1.2.1.1.5.0"
	oidSysDescr        = "1.3.6.1.2.1.1.1.0"
	oidSysObjectID     = "1.3.6.1.2.1.1.2.0"
	oidLLDPSysName     = "1.0.8802.1.1.2.1.4.1.1.9"
	oidLLDPPort        = "1.0.8802.1.1.2.1.4.1.1.7"
	oidBridgeMAC       = "1.3.6.1.2.1.17.4.3.1.1"
	oidBridgePort      = "1.3.6.1.2.1.17.4.3.1.2"
	defaultSNMPTimeout = 2 * time.Second
)

// SNMPRunner polls devices for LLDP/CDP/Bridge data.
type SNMPRunner struct {
	communities []string
	timeout     time.Duration
	log         Logger
	newClient   snmpFactory
}

func NewSNMPRunner(log Logger, communities []string) *SNMPRunner {
	if len(communities) == 0 {
		communities = []string{"public", "private"}
	}
	return &SNMPRunner{communities: communities, timeout: defaultSNMPTimeout, log: log, newClient: defaultSNMPFactory}
}

func (r *SNMPRunner) Name() string { return "snmp" }

func (r *SNMPRunner) Run(job Job, recorder Recorder) error {
	if job.CIDR == nil || recorder == nil {
		return nil
	}
	ctx := context.Background()
	for _, ip := range hostsFromCIDR(job.CIDR) {
		r.pollTarget(ctx, ip, recorder)
	}
	return nil
}

func (r *SNMPRunner) pollTarget(ctx context.Context, ip net.IP, recorder Recorder) {
	target := ip.String()
	for _, community := range r.communities {
		client, err := r.newClient(target, community)
		if err != nil {
			r.log.Error("snmp connect", "ip", target, "err", err)
			continue
		}
		r.collectDevice(ctx, client, target, recorder)
		client.Close()
		break
	}
}

func (r *SNMPRunner) collectDevice(ctx context.Context, client snmpClient, target string, recorder Recorder) {
	pkt, err := client.Get([]string{oidSysName, oidSysDescr, oidSysObjectID})
	if err != nil {
		r.log.Error("snmp get", "ip", target, "err", err)
		return
	}
	hostname := snmpString(pkt, oidSysName)
	descr := snmpString(pkt, oidSysDescr)
	sysObject := snmpString(pkt, oidSysObjectID)
	vendor := vendorFromDescr(descr)
	recorder.RecordDevice(ctx, DeviceObservation{
		ID:          target,
		IPv4:        target,
		Hostname:    hostname,
		Vendor:      vendor,
		SysObjectID: sysObject,
		TypeHint:    classifyBySysObject(sysObject),
		Source:      r.Name(),
		Confidence:  0.8,
	})
	r.collectLLDP(ctx, client, target, recorder)
	r.collectBridge(ctx, client, target, recorder)
}

func (r *SNMPRunner) collectLLDP(ctx context.Context, client snmpClient, localID string, recorder Recorder) {
	err := client.Walk(oidLLDPSysName, func(pdu gosnmp.SnmpPDU) error {
		neighbor := sanitizeName(fmt.Sprintf("%v", pdu.Value))
		if neighbor == "" {
			return nil
		}
		recorder.RecordDevice(ctx, DeviceObservation{
			ID:         neighbor,
			Hostname:   neighbor,
			Source:     "lldp",
			TypeHint:   "switch",
			Confidence: 0.4,
		})
		recorder.RecordLink(ctx, LinkObservation{
			ADevice:    localID,
			BDevice:    neighbor,
			Kind:       "lldp",
			Source:     r.Name(),
			Confidence: 0.9,
		})
		return nil
	})
	if err != nil {
		r.log.Warn("lldp walk", "ip", localID, "err", err)
	}
}

func (r *SNMPRunner) collectBridge(ctx context.Context, client snmpClient, localID string, recorder Recorder) {
	macs := map[string]string{}
	err := client.Walk(oidBridgeMAC, func(pdu gosnmp.SnmpPDU) error {
		if bytes, ok := pdu.Value.([]byte); ok {
			macs[pdu.Name] = macFromBytes(bytes)
		}
		return nil
	})
	if err != nil {
		r.log.Warn("bridge mac walk", "ip", localID, "err", err)
		return
	}
	err = client.Walk(oidBridgePort, func(pdu gosnmp.SnmpPDU) error {
		mac := macs[strings.Replace(pdu.Name, oidBridgePort, oidBridgeMAC, 1)]
		if mac == "" {
			return nil
		}
		recorder.RecordDevice(ctx, DeviceObservation{ID: mac, MAC: mac, Source: "bridge", TypeHint: "endpoint"})
		recorder.RecordLink(ctx, LinkObservation{
			ADevice:    localID,
			BDevice:    mac,
			Kind:       "bridge",
			Source:     r.Name(),
			Confidence: 0.5,
		})
		return nil
	})
	if err != nil {
		r.log.Warn("bridge port walk", "ip", localID, "err", err)
	}
}

func snmpString(pkt *gosnmp.SnmpPacket, oid string) string {
	if pkt == nil {
		return ""
	}
	for _, v := range pkt.Variables {
		if v.Name == oid {
			switch val := v.Value.(type) {
			case string:
				return strings.TrimSpace(val)
			case []byte:
				return strings.TrimSpace(string(val))
			}
		}
	}
	return ""
}

func vendorFromDescr(descr string) string {
	if descr == "" {
		return ""
	}
	parts := strings.Split(descr, " ")
	if len(parts) > 0 {
		return parts[0]
	}
	return descr
}

func classifyBySysObject(oid string) string {
	switch {
	case strings.Contains(oid, "9.1"):
		return "router"
	case strings.Contains(oid, "11.2.3"):
		return "switch"
	default:
		return "device"
	}
}
