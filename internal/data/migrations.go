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
            targets TEXT,
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
	if err := ensureColumn(db, "scans", "targets", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn(db, "devices", "hostname", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn(db, "devices", "sys_object_id", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn(db, "links", "kind", "TEXT"); err != nil {
		return err
	}
	return nil
}

func ensureColumn(db *sql.DB, table, column, columnType string) error {
	query := fmt.Sprintf(`PRAGMA table_info("%s");`, table)
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("scan table info %s: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate columns %s: %w", table, err)
	}
	stmt := fmt.Sprintf(`ALTER TABLE "%s" ADD COLUMN "%s" %s;`, table, column, columnType)
	if _, err := db.Exec(stmt); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
	}
	return nil
}
