package scanner

import (
	"context"
	"net"
	"testing"

	"github.com/tomfocker/lanmapper/internal/logger"
)

type stubRecorder struct {
	lastTargets []string
}

func (s *stubRecorder) InsertScan(ctx context.Context, scanID string, targets []string) error {
	s.lastTargets = targets
	return nil
}
func (s *stubRecorder) FinishScan(ctx context.Context, scanID string, status string) error {
	return nil
}

func TestSchedulerTrigger(t *testing.T) {
	mgr := &Manager{
		jobs: make(chan Job, 4),
		log:  loggerLForTest(),
	}
	rec := &stubRecorder{}
	sched := NewScheduler(mgr, rec)

	_, cidr, _ := net.ParseCIDR("192.168.1.0/24")
	targets := []DetectedCIDR{{CIDR: cidr, Interface: "eth0"}}
	if _, err := sched.Trigger(context.Background(), targets); err != nil {
		t.Fatalf("trigger failed: %v", err)
	}
	select {
	case job := <-mgr.jobs:
		if job.Interface != "eth0" {
			t.Fatalf("unexpected interface %s", job.Interface)
		}
	default:
		t.Fatal("job was not enqueued")
	}
	if len(rec.lastTargets) != 1 || rec.lastTargets[0] != "192.168.1.0/24" {
		t.Fatalf("recorder not notified: %#v", rec.lastTargets)
	}
}

func TestMergeTargets(t *testing.T) {
	auto := []DetectedCIDR{{
		CIDR:      &net.IPNet{IP: net.IPv4(10, 0, 0, 0), Mask: net.CIDRMask(24, 32)},
		Interface: "eth0",
	}}
	cfg := []string{"192.168.1.0/24", " 172.16.0.0/16 "}
	res, err := MergeTargets(auto, cfg)
	if err != nil {
		t.Fatalf("merge failed: %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(res))
	}
	if res[1].Interface != "eth0" {
		t.Fatalf("configured target missing interface fallback")
	}
}

// loggerLForTest returns a no-op logger to satisfy dependencies.
func loggerLForTest() Logger {
	return logger.L()
}
