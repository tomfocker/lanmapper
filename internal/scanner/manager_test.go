package scanner

import (
	"context"
	"net"
	"testing"
)

type mockRunner struct {
	name string
	runs int
}

func (m *mockRunner) Name() string { return m.name }

func (m *mockRunner) Run(job Job, recorder Recorder) error {
	m.runs++
	return nil
}

func TestManagerDispatchesJobs(t *testing.T) {
	r1 := &mockRunner{name: "r1"}
	r2 := &mockRunner{name: "r2"}
	rec := &managerStubRecorder{}
	mgr := NewManager(rec, r1, r2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	_, cidr, _ := net.ParseCIDR("192.168.1.0/30")
	mgr.Enqueue(Job{CIDR: cidr})
	mgr.Stop()
	if r1.runs == 0 || r2.runs == 0 {
		t.Fatalf("runners not executed: r1=%d r2=%d", r1.runs, r2.runs)
	}
}

type managerStubRecorder struct{}

func (s *managerStubRecorder) RecordDevice(context.Context, DeviceObservation) {}
func (s *managerStubRecorder) RecordLink(context.Context, LinkObservation)     {}
func (s *managerStubRecorder) SetGateway(string)                               {}
func (s *managerStubRecorder) Close()                                         {}
