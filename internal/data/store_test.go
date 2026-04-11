package data

import (
	"context"
	"database/sql"
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

func TestInsertAndFinishScan(t *testing.T) {
	dir := t.TempDir()
	store, err := New(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	ctx := context.Background()
	if err := store.InsertScan(ctx, "scan1", []string{"192.168.1.0/24"}); err != nil {
		t.Fatalf("insert scan: %v", err)
	}
	var status, targets string
	var finished sql.NullTime
	if err := store.db.QueryRowContext(ctx, `SELECT status, targets, finished_at FROM scans WHERE id = ?`, "scan1").
		Scan(&status, &targets, &finished); err != nil {
		t.Fatalf("query scan: %v", err)
	}
	if status != "running" {
		t.Fatalf("unexpected status %s", status)
	}
	if targets != `["192.168.1.0/24"]` {
		t.Fatalf("unexpected targets %s", targets)
	}
	if finished.Valid {
		t.Fatalf("expected unfinished scan")
	}
	if err := store.FinishScan(ctx, "scan1", "done"); err != nil {
		t.Fatalf("finish scan: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT status, finished_at FROM scans WHERE id = ?`, "scan1").
		Scan(&status, &finished); err != nil {
		t.Fatalf("query scan after finish: %v", err)
	}
	if status != "done" {
		t.Fatalf("status not updated: %s", status)
	}
	if !finished.Valid {
		t.Fatalf("finished_at not set")
	}
}
