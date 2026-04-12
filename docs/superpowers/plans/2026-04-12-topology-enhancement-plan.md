# Topology Enhancement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Enrich device records with names/types and populate link data using SNMP (LLDP/Bridge), OUI, and passive heuristics so the UI shows meaningful topology.

**Architecture:** Extend discovery observations/recorder to carry vendor/type/link kind metadata, embed a compact OUI lookup, upgrade the SNMP runner to query LLDP + Bridge tables, and add a passive gateway fallback. Recorder writes enriched devices/links into SQLite; topology builder/UI read existing APIs with richer data.

**Tech Stack:** Go 1.25, gosnmp, embedded CSV assets, SQLite, Fiber API.

---

### Task 1: Extend Observation & Data Schema

**Files:**
- Modify: `internal/scanner/recorder.go`
- Modify: `internal/scanner/recorder_test.go`
- Modify: `internal/models/device.go` (if fields missing)
- Modify: `internal/data/migrations.go`
- Modify: `internal/data/store.go`

- [ ] **Step 1:** Update structs in `recorder.go` to include new fields

```go
type DeviceObservation struct {
    ...
    SysObjectID string
    Services    []string
    TypeHint    string
}

type LinkObservation struct {
    ...
    Kind string // e.g. "lldp", "bridge", "gateway"
}
```

- [ ] **Step 2:** Add `kind` column to `links` table via migration helper

```go
ALTER TABLE links ADD COLUMN kind TEXT DEFAULT '';
```

- [ ] **Step 3:** Ensure `models.Link` / `models.Device` expose `Kind`/`Hostname`/`Type` fields (add if missing) and `data.Store.UpsertLink/UpsertDevice` writes them.

- [ ] **Step 4:** Extend recorder handlers to merge new fields, set `Type` when `TypeHint` present, and persist `Kind` for links.

- [ ] **Step 5:** Update recorder tests to cover new merge behavior (e.g., TypeHint applied, link kind stored).

### Task 2: Embed OUI Lookup & Passive Heuristics

**Files:**
- Create: `internal/scanner/oui/oui.csv` (trimmed dataset)
- Create: `internal/scanner/oui/oui.go`
- Modify: `go:embed` directives if needed
- Modify: `internal/scanner/recorder.go`
- Modify: `README.md`, `docs/usage.md`

- [ ] **Step 1:** Add compressed OUI CSV (prefix,vendor,typeHint) under `internal/scanner/oui/`.

- [ ] **Step 2:** Implement helper:

```go
type OUILookup struct { data map[string]VendorHint }
func (l *OUILookup) Lookup(mac string) VendorHint {...}
```

- [ ] **Step 3:** Initialize lookup in `recorder.go` (via `sync.Once`), use it to fill `Vendor`/`Type` when MAC observed.

- [ ] **Step 4:** Add passive gateway inference helper (e.g., mark first detected default gateway IP as `router`).

- [ ] **Step 5:** Update docs to describe OUI source and how to refresh it.

### Task 3: SNMP Runner LLDP/Bridge Implementation

**Files:**
- Modify: `internal/scanner/protocol_snmp.go`
- Create: `internal/scanner/snmp_helpers.go`
- Modify: `internal/scanner/protocol_snmp_test.go`

- [ ] **Step 1:** Wrap gosnmp interactions in helper struct (`snmpClient`) to ease testing.

- [ ] **Step 2:** Implement LLDP fetch:

```go
func (r *SNMPRunner) fetchLLDP(client snmpClient, recorder Recorder, localMAC string)
```
 Map neighbor entries to `LinkObservation{Kind:"lldp"}` & `DeviceObservation`.

- [ ] **Step 3:** Implement Bridge FDB fetch to map MAC→port; emit `LinkObservation{Kind:"bridge"}` between local device and each MAC.

- [ ] **Step 4:** Populate `DeviceObservation` with `Hostname` (sysName), `SysObjectID`, `TypeHint` (based on known vendor OIDs).

- [ ] **Step 5:** Add unit tests using fake snmpClient returning scripted PDUs verifying observations are produced.

### Task 4: Passive Gateway Link Fallback

**Files:**
- Modify: `internal/scanner/interfaces.go`
- Modify: `internal/scanner/recorder.go`
- Modify: `internal/topology/builder.go`
- Modify: `internal/topology/builder_test.go`

- [ ] **Step 1:** Detect default gateway IP/interface (most logic already available) and synthesize `LinkObservation{Kind:"gateway"}` connecting each device lacking explicit link to gateway.

- [ ] **Step 2:** In topology builder, surface `link.Kind` for UI (maybe convert to weight/confidence if needed).

- [ ] **Step 3:** Update topology tests to assert at least one gateway link is returned when no real links exist.

### Task 5: Wire Everything & Validate

**Files/Commands:**
- Modify: `cmd/lanmapper/main.go`
- Modify: `internal/api/routes_test.go` (if schema changes)
- Run: `go test ./...`, `docker build -t lanmapper .`

- [ ] **Step 1:** Ensure OUI lookup/SNMP runner initialized in main.
- [ ] **Step 2:** Run full test suite.
- [ ] **Step 3:** Build docker image to verify compile.

### Task 6: Documentation Updates

**Files:**
- Modify: `README.md`
- Modify: `docs/usage.md`

- [ ] **Step 1:** Document SNMP requirements (communities, enabling LLDP) and how to provide custom OUI dataset.
- [ ] **Step 2:** Add troubleshooting section explaining fallback gateway links vs. LLDP links.

---
