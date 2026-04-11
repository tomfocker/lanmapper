package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/tomfocker/lanmapper/internal/scanner"
)

func TestScanRouteDefaultTargets(t *testing.T) {
	app := fiber.New()
	mgr := scanner.NewManager()
	rec := &stubRecorder{}
	sched := scanner.NewScheduler(mgr, rec)
	_, cidr, _ := net.ParseCIDR("192.168.1.0/24")
	defaultTargets := []scanner.DetectedCIDR{{CIDR: cidr, Interface: "eth0"}}

	RegisterRoutes(app, nil, nil, mgr, nil, sched, defaultTargets)

	req := httptest.NewRequest("POST", "/scans", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	var payload struct {
		Targets []string `json:"targets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Targets) != 1 || payload.Targets[0] != "192.168.1.0/24" {
		t.Fatalf("unexpected targets payload: %+v", payload.Targets)
	}
	if len(rec.lastTargets) != 1 || rec.lastTargets[0] != "192.168.1.0/24" {
		t.Fatalf("scheduler not triggered: %+v", rec.lastTargets)
	}
}

func TestScanRouteInvalidCIDR(t *testing.T) {
	app := fiber.New()
	mgr := scanner.NewManager()
	rec := &stubRecorder{}
	sched := scanner.NewScheduler(mgr, rec)
	RegisterRoutes(app, nil, nil, mgr, nil, sched, nil)

	body := bytes.NewBufferString(`{"cidr":["badcidr"]}`)
	req := httptest.NewRequest("POST", "/scans", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 got %d", resp.StatusCode)
	}
	if len(rec.lastTargets) != 0 {
		t.Fatalf("scheduler should not run on invalid cidr")
	}
}

func TestScanRouteSchedulerUnavailable(t *testing.T) {
	app := fiber.New()
	RegisterRoutes(app, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest("POST", "/scans", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 503 {
		t.Fatalf("expected 503 got %d", resp.StatusCode)
	}
}

func TestScanRouteCustomCIDR(t *testing.T) {
	app := fiber.New()
	mgr := scanner.NewManager()
	rec := &stubRecorder{}
	sched := scanner.NewScheduler(mgr, rec)
	_, cidr, _ := net.ParseCIDR("10.0.0.0/8")
	defaultTargets := []scanner.DetectedCIDR{{CIDR: cidr, Interface: "eth1"}}
	RegisterRoutes(app, nil, nil, mgr, nil, sched, defaultTargets)

	body := bytes.NewBufferString(`{"cidr":["192.168.1.5/32"],"interface":"eth9"}`)
	req := httptest.NewRequest("POST", "/scans", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	if len(rec.lastTargets) != 1 || rec.lastTargets[0] != "192.168.1.0/24" {
		t.Fatalf("expected canonicalized cidr, got %+v", rec.lastTargets)
	}
}

type stubRecorder struct {
	lastTargets []string
}

func (s *stubRecorder) InsertScan(ctx context.Context, scanID string, targets []string) error {
	s.lastTargets = append([]string{}, targets...)
	return nil
}

func (s *stubRecorder) FinishScan(ctx context.Context, scanID string, status string) error {
	return nil
}
