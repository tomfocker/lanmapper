package scanner

import (
	"context"
	"testing"

	"github.com/gosnmp/gosnmp"
)

type stubSNMPClient struct {
	packet     *gosnmp.SnmpPacket
	lldpPDUs   []gosnmp.SnmpPDU
	bridgeMAC  []gosnmp.SnmpPDU
	bridgePort []gosnmp.SnmpPDU
}

func (s *stubSNMPClient) Get(oids []string) (*gosnmp.SnmpPacket, error) {
	return s.packet, nil
}

func (s *stubSNMPClient) Walk(oid string, walkFn gosnmp.WalkFunc) error {
	var list []gosnmp.SnmpPDU
	switch oid {
	case oidLLDPSysName:
		list = s.lldpPDUs
	case oidBridgeMAC:
		list = s.bridgeMAC
	case oidBridgePort:
		list = s.bridgePort
	default:
		list = nil
	}
	for _, p := range list {
		if err := walkFn(p); err != nil {
			return err
		}
	}
	return nil
}

func (s *stubSNMPClient) Close() {}

type recorderSpy struct {
	devices []DeviceObservation
	links   []LinkObservation
}

func (r *recorderSpy) RecordDevice(_ context.Context, obs DeviceObservation) {
	r.devices = append(r.devices, obs)
}
func (r *recorderSpy) RecordLink(_ context.Context, obs LinkObservation) {
	r.links = append(r.links, obs)
}
func (r *recorderSpy) SetGateway(string) {}
func (r *recorderSpy) Close()            {}

func TestSNMPRunnerCollectsLLDPAndBridge(t *testing.T) {
	runner := NewSNMPRunner(loggerLForTest(), []string{"public"})
	client := &stubSNMPClient{
		packet: &gosnmp.SnmpPacket{Variables: []gosnmp.SnmpPDU{
			{Name: oidSysName, Value: "core-sw"},
			{Name: oidSysDescr, Value: "Cisco Switch"},
			{Name: oidSysObjectID, Value: ".1.3.6.1.4.1.9.1.100"},
		}},
		lldpPDUs:   []gosnmp.SnmpPDU{{Name: oidLLDPSysName + ".1.1", Value: "Edge-AP"}},
		bridgeMAC:  []gosnmp.SnmpPDU{{Name: oidBridgeMAC + ".1.1", Value: []byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}}},
		bridgePort: []gosnmp.SnmpPDU{{Name: oidBridgePort + ".1.1", Value: int(1)}},
	}
	runner.newClient = func(target, community string) (snmpClient, error) {
		return client, nil
	}
	rec := &recorderSpy{}
	runner.collectDevice(context.Background(), client, "192.168.1.10", rec)
	if len(rec.devices) == 0 {
		t.Fatalf("expected devices from SNMP")
	}
	if rec.devices[0].TypeHint == "" {
		t.Fatalf("type hint not set")
	}
	if len(rec.links) == 0 {
		t.Fatalf("expected links from SNMP")
	}
	foundLLDP := false
	for _, link := range rec.links {
		if link.Kind == "lldp" {
			foundLLDP = true
		}
	}
	if !foundLLDP {
		t.Fatalf("lldp link missing: %+v", rec.links)
	}
}
