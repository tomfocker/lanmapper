package data

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tomfocker/lanmapper/internal/models"
)

func TestUpsertAndListDevices(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	ctx := context.Background()
	if err := store.UpsertDevice(ctx, &models.Device{ID: "dev1", IPv4: "192.168.1.10"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	list, err := store.ListDevices(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].ID != "dev1" {
		t.Fatalf("unexpected devices: %+v", list)
	}
}

func TestUpsertAndListLinks(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	ctx := context.Background()
	link := &models.Link{ID: "link1", ADevice: "dev1", AInterface: "eth0", BDevice: "dev2", BInterface: "eth0"}
	if err := store.UpsertLink(ctx, link); err != nil {
		t.Fatalf("upsert link: %v", err)
	}
	links, err := store.ListLinks(ctx)
	if err != nil {
		t.Fatalf("list links: %v", err)
	}
	if len(links) != 1 || links[0].ID != "link1" {
		t.Fatalf("unexpected links: %+v", links)
	}
}

func TestMigrateIdempotent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	for i := 0; i < 2; i++ {
		store, err := New(path)
		if err != nil {
			t.Fatalf("open store: %v", err)
		}
		store.db.Close()
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("db not created: %v", err)
	}
}
