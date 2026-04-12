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
