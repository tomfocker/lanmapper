package data

import (
	"database/sql"
	"fmt"
)

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS devices (
            id TEXT PRIMARY KEY,
            ipv4 TEXT,
            ipv6 TEXT,
            mac TEXT,
            vendor TEXT,
            type TEXT,
            last_seen DATETIME,
            confidence REAL
        );`,
		`CREATE TABLE IF NOT EXISTS interfaces (
            id TEXT PRIMARY KEY,
            device_id TEXT,
            name TEXT,
            mac TEXT,
            speed_mbps INTEGER,
            vlan TEXT,
            is_uplink INTEGER,
            FOREIGN KEY(device_id) REFERENCES devices(id)
        );`,
		`CREATE TABLE IF NOT EXISTS links (
            id TEXT PRIMARY KEY,
            a_device TEXT,
            a_interface TEXT,
            b_device TEXT,
            b_interface TEXT,
            media TEXT,
            speed_mbps INTEGER,
            source TEXT,
            confidence REAL
        );`,
		`CREATE TABLE IF NOT EXISTS scans (
            id TEXT PRIMARY KEY,
            status TEXT,
            started_at DATETIME,
            finished_at DATETIME,
            config_snapshot TEXT,
            stats TEXT
        );`,
		`CREATE TABLE IF NOT EXISTS reports (
            id TEXT PRIMARY KEY,
            scan_id TEXT,
            type TEXT,
            path TEXT,
            created_at DATETIME,
            FOREIGN KEY(scan_id) REFERENCES scans(id)
        );`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}
