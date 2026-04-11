package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"

	"github.com/tomfocker/lanmapper/internal/models"
)

// Store provides persistence access over SQLite.
type Store struct {
	db *sql.DB
}

// New opens (and migrates) database at path.
func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetConnMaxLifetime(0)
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// UpsertDevice inserts or updates device metadata.
func (s *Store) UpsertDevice(ctx context.Context, d *models.Device) error {
	if d.LastSeen.IsZero() {
		d.LastSeen = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO devices
        (id, ipv4, ipv6, mac, vendor, type, last_seen, confidence)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
          ipv4=excluded.ipv4,
          ipv6=excluded.ipv6,
          mac=excluded.mac,
          vendor=excluded.vendor,
          type=excluded.type,
          last_seen=excluded.last_seen,
          confidence=excluded.confidence`,
		d.ID, d.IPv4, d.IPv6, d.MAC, d.Vendor, d.Type, d.LastSeen, d.Confidence,
	)
	if err != nil {
		return fmt.Errorf("upsert device: %w", err)
	}
	return nil
}

// ListDevices returns all devices ordered by last seen desc.
func (s *Store) ListDevices(ctx context.Context) ([]models.Device, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, ipv4, ipv6, mac, vendor, type, last_seen, confidence FROM devices ORDER BY last_seen DESC`)
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}
	defer rows.Close()
	var out []models.Device
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.ID, &d.IPv4, &d.IPv6, &d.MAC, &d.Vendor, &d.Type, &d.LastSeen, &d.Confidence); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// UpsertLink stores link edges.
func (s *Store) UpsertLink(ctx context.Context, l *models.Link) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO links
        (id, a_device, a_interface, b_device, b_interface, media, speed_mbps, source, confidence)
        VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
          a_device=excluded.a_device,
          a_interface=excluded.a_interface,
          b_device=excluded.b_device,
          b_interface=excluded.b_interface,
          media=excluded.media,
          speed_mbps=excluded.speed_mbps,
          source=excluded.source,
          confidence=excluded.confidence`,
		l.ID, l.ADevice, l.AInterface, l.BDevice, l.BInterface, l.Media, l.SpeedMbps, l.Source, l.Confidence,
	)
	if err != nil {
		return fmt.Errorf("upsert link: %w", err)
	}
	return nil
}

// ListLinks returns current link snapshot.
func (s *Store) ListLinks(ctx context.Context) ([]models.Link, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, a_device, a_interface, b_device, b_interface, media, speed_mbps, source, confidence FROM links`)
	if err != nil {
		return nil, fmt.Errorf("query links: %w", err)
	}
	defer rows.Close()
	var out []models.Link
	for rows.Next() {
		var l models.Link
		if err := rows.Scan(&l.ID, &l.ADevice, &l.AInterface, &l.BDevice, &l.BInterface, &l.Media, &l.SpeedMbps, &l.Source, &l.Confidence); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// InsertScan creates a new scan record with running status and JSON targets.
func (s *Store) InsertScan(ctx context.Context, scanID string, targets []string) error {
	if targets == nil {
		targets = []string{}
	}
	payload, err := json.Marshal(targets)
	if err != nil {
		return fmt.Errorf("marshal targets: %w", err)
	}
	started := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `INSERT INTO scans (id, status, started_at, targets) VALUES (?, ?, ?, ?)`,
		scanID, "running", started, string(payload)); err != nil {
		return fmt.Errorf("insert scan: %w", err)
	}
	return nil
}

// FinishScan updates scan status and finished timestamp.
func (s *Store) FinishScan(ctx context.Context, scanID string, status string) error {
	if status == "" {
		status = "finished"
	}
	finished := time.Now().UTC()
	res, err := s.db.ExecContext(ctx, `UPDATE scans SET status = ?, finished_at = ? WHERE id = ?`, status, finished, scanID)
	if err != nil {
		return fmt.Errorf("finish scan: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("finish scan rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("finish scan: id %s not found", scanID)
	}
	return nil
}
