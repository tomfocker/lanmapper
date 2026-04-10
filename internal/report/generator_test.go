package report

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tomfocker/lanmapper/internal/data"
	"github.com/tomfocker/lanmapper/internal/models"
)

func setupStore(t *testing.T) *data.Store {
	t.Helper()
	dir := t.TempDir()
	store, err := data.New(filepath.Join(dir, "report.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	ctx := context.Background()
	store.UpsertDevice(ctx, &models.Device{ID: "dev1"})
	return store
}

func TestExportJSON(t *testing.T) {
	store := setupStore(t)
	gen := NewGenerator(store, t.TempDir())
	path, err := gen.ExportJSON(context.Background())
	if err != nil {
		t.Fatalf("export json: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file missing: %v", err)
	}
}

func TestExportCSV(t *testing.T) {
	store := setupStore(t)
	gen := NewGenerator(store, t.TempDir())
	path, err := gen.ExportCSV(context.Background())
	if err != nil {
		t.Fatalf("export csv: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file missing: %v", err)
	}
}
