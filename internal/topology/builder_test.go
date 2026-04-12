package topology

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/tomfocker/lanmapper/internal/data"
	"github.com/tomfocker/lanmapper/internal/models"
)

func TestBuilderRebuild(t *testing.T) {
	dir := t.TempDir()
	store, err := data.New(filepath.Join(dir, "graph.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	ctx := context.Background()
	if err := store.UpsertDevice(ctx, &models.Device{ID: "dev1"}); err != nil {
		t.Fatalf("device: %v", err)
	}
	if err := store.UpsertLink(ctx, &models.Link{ID: "link1", ADevice: "dev1", Kind: "gateway"}); err != nil {
		t.Fatalf("link: %v", err)
	}
	builder := NewBuilder(store)
	devices, links, err := builder.Rebuild(ctx)
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}
	if len(devices) != 1 || len(links) != 1 {
		t.Fatalf("unexpected graph: %v %v", devices, links)
	}
	if links[0].Kind != "gateway" {
		t.Fatalf("link kind not preserved: %+v", links[0])
	}
}
