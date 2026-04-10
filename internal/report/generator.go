package report

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"github.com/tomfocker/lanmapper/internal/data"
	"github.com/tomfocker/lanmapper/internal/models"
)

// Generator handles report exports.
type Generator struct {
	store  *data.Store
	outDir string
}

func NewGenerator(store *data.Store, outDir string) *Generator {
	return &Generator{store: store, outDir: outDir}
}

func (g *Generator) ExportJSON(ctx context.Context) (string, error) {
	devices, err := g.store.ListDevices(ctx)
	if err != nil {
		return "", err
	}
	links, err := g.store.ListLinks(ctx)
	if err != nil {
		return "", err
	}
	payload := struct {
		Generated time.Time       `json:"generated"`
		Devices   []models.Device `json:"devices"`
		Links     []models.Link   `json:"links"`
	}{Generated: time.Now().UTC(), Devices: devices, Links: links}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(g.outDir, fmt.Sprintf("report-%s.json", uuid.NewString()))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (g *Generator) ExportCSV(ctx context.Context) (string, error) {
	devices, err := g.store.ListDevices(ctx)
	if err != nil {
		return "", err
	}
	buf := bytes.Buffer{}
	writer := csv.NewWriter(&buf)
	writer.Write([]string{"ID", "IPv4", "IPv6", "MAC", "Vendor", "Type", "LastSeen"})
	for _, d := range devices {
		writer.Write([]string{d.ID, d.IPv4, d.IPv6, d.MAC, d.Vendor, d.Type, d.LastSeen.Format(time.RFC3339)})
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	path := filepath.Join(g.outDir, fmt.Sprintf("devices-%s.csv", uuid.NewString()))
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
