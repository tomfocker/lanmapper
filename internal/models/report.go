package models

import "time"

// Report represents an exported artifact for a scan.
type Report struct {
	ID      string    `db:"id" json:"id"`
	ScanID  string    `db:"scan_id" json:"scan_id"`
	Type    string    `db:"type" json:"type"`
	Path    string    `db:"path" json:"path"`
	Created time.Time `db:"created_at" json:"created_at"`
}
