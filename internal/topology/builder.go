package topology

import (
	"context"

	"github.com/tomfocker/lanmapper/internal/data"
	"github.com/tomfocker/lanmapper/internal/models"
)

// Edge represents a link with confidence.
type Edge struct {
	LinkID     string      `json:"link_id"`
	Source     string      `json:"source"`
	Confidence float64     `json:"confidence"`
	A          models.Link `json:"a"`
}

// Builder composes data into derived topology.
type Builder struct {
	store *data.Store
}

func NewBuilder(store *data.Store) *Builder {
	return &Builder{store: store}
}

// Rebuild loads devices/links and emits aggregated graph (placeholder).
func (b *Builder) Rebuild(ctx context.Context) ([]models.Device, []models.Link, error) {
	devices, err := b.store.ListDevices(ctx)
	if err != nil {
		return nil, nil, err
	}
	links, err := b.store.ListLinks(ctx)
	if err != nil {
		return nil, nil, err
	}
	return devices, links, nil
}
