package models

import "time"

// Scan tracks execution metadata.
type Scan struct {
	ID         string    `db:"id" json:"id"`
	Status     string    `db:"status" json:"status"`
	StartedAt  time.Time `db:"started_at" json:"started_at"`
	FinishedAt time.Time `db:"finished_at" json:"finished_at"`
	Config     string    `db:"config_snapshot" json:"config_snapshot"`
	Stats      string    `db:"stats" json:"stats"`
}
