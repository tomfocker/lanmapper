package scanner

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/tomfocker/lanmapper/internal/data"
)

func TestRecorderStoresDeviceOnceWithinTTL(t *testing.T) {
	dir := t.TempDir()
	store, err := data.New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	rec := NewRecorder(store, loggerLForTest())
	recImpl := rec.(*recorder)
	recImpl.cacheTTL = 50 * time.Millisecond

	ctx := context.Background()
	rec.RecordDevice(ctx, DeviceObservation{ID: "aa:bb:cc:dd:ee:ff", IPv4: "192.168.1.2"})
	rec.RecordDevice(ctx, DeviceObservation{ID: "aa:bb:cc:dd:ee:ff", IPv4: "192.168.1.2"})
	time.Sleep(100 * time.Millisecond)
	rec.RecordDevice(ctx, DeviceObservation{ID: "aa:bb:cc:dd:ee:ff", IPv4: "192.168.1.2"})
	rec.Close()

	devices, err := store.ListDevices(ctx)
	if err != nil {
		t.Fatalf("list devices: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device got %d", len(devices))
	}
	if devices[0].ID != "aa:bb:cc:dd:ee:ff" {
		t.Fatalf("unexpected device id %s", devices[0].ID)
	}
}

func TestRecorderAppliesTypeHintAndLinkKind(t *testing.T) {
	dir := t.TempDir()
	store, err := data.New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	rec := NewRecorder(store, loggerLForTest())
	ctx := context.Background()
	rec.RecordDevice(ctx, DeviceObservation{
		ID:       "aa:bb:cc:dd:ee:11",
		TypeHint: "switch",
		Hostname: "sw1",
	})
	rec.RecordLink(ctx, LinkObservation{
		ADevice: "aa:bb:cc:dd:ee:11",
		BDevice: "bb:cc:dd:ee:ff:00",
		Kind:    "lldp",
	})
	time.Sleep(20 * time.Millisecond)
	rec.Close()

	devs, err := store.ListDevices(ctx)
	if err != nil {
		t.Fatalf("list devices: %v", err)
	}
	if devs[0].Type != "switch" || devs[0].Hostname != "sw1" {
		t.Fatalf("device not enriched: %+v", devs[0])
	}
	links, err := store.ListLinks(ctx)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if links[0].Kind != "lldp" {
		t.Fatalf("link kind missing: %+v", links[0])
	}
}

func TestRecorderGatewayFallbackLink(t *testing.T) {
	dir := t.TempDir()
	store, err := data.New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	rec := NewRecorder(store, loggerLForTest())
	rec.SetGateway("192.168.1.1")
	ctx := context.Background()
	rec.RecordDevice(ctx, DeviceObservation{
		ID:     "192.168.1.10",
		IPv4:   "192.168.1.10",
		Source: "arp_nd",
	})
	time.Sleep(20 * time.Millisecond)
	rec.Close()

	links, err := store.ListLinks(ctx)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) == 0 || links[0].Kind != "gateway" {
		t.Fatalf("gateway link not created: %+v", links)
	}
}

func TestRecorderCloseStopsWorkers(t *testing.T) {
	dir := t.TempDir()
	store, err := data.New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	rec := NewRecorder(store, loggerLForTest())
	done := make(chan struct{})
	go func() {
		rec.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("recorder close did not finish")
	}
}
