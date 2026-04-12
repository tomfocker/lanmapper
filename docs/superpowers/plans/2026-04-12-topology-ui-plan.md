# Topology UI Enhancement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show device names/types/vendors and link kinds in the web UI so users can quickly understand topology, while keeping data in sync with the enriched backend.

**Architecture:** Reuse existing REST endpoints, map `device.hostname/vendor/type` to UI-friendly chips/node labels, expose `link.kind` in topology graph, and add legend/tooltips. Frontend stays in React + D3 setup; backend changes limited to APIs if needed (currently sufficient).

**Tech Stack:** React + TypeScript + Vite, CSS Modules, existing topology rendering utilities.

---

### Task 1: Update API Client Types

**Files:**
- Modify: `webapp/src/api/types.ts`
- Modify: `webapp/src/api/client.ts`

- [ ] **Step 1:** Extend `Device` and `Link` interfaces

```ts
export interface Device {
  id: string
  ipv4: string
  mac: string
  vendor?: string
  type?: string
  hostname?: string
  sys_object_id?: string
  last_seen: string
}

export interface Link {
  id: string
  a_device: string
  a_interface?: string
  b_device: string
  b_interface?: string
  kind?: string
  confidence: number
}
```

- [ ] **Step 2:** Ensure fetch helpers (`fetchDevices`, `fetchLinks`) map JSON to these types (likely already automatic, but update generics).

### Task 2: Device List Cards

**Files:**
- Modify: `webapp/src/components/DeviceList.tsx`
- Modify: `webapp/src/App.css` (or dedicated CSS)

- [ ] **Step 1:** Render device title as `hostname ?? vendor ?? id`, subtitle `type ?? "endpoint"`, with IP displayed below.

```tsx
const title = device.hostname || device.vendor || device.id
const subtitle = device.type || "endpoint"
```

- [ ] **Step 2:** Add small badge for vendor/type (e.g., `<span className="device-chip">{subtitle}</span>`), update CSS for chips.

- [ ] **Step 3:** Show `last_seen` as relative time (use `new Date(device.last_seen).toLocaleString()` for now).

### Task 3: Topology Graph Node Formatting

**Files:**
- Modify: `webapp/src/components/TopologyGraph.tsx`
- Modify: any helper that builds node labels

- [ ] **Step 1:** When constructing nodes, set `label` to multiline string `"${name} (${type})\n${device.ipv4 || device.id}"` where `name = device.hostname || device.vendor || device.id`.

- [ ] **Step 2:** Apply CSS classes based on type (router/switch/endpoint) for fill color.

```ts
const nodeClass = `node node-${(device.type || 'endpoint').toLowerCase()}`
```

- [ ] **Step 3:** Update stylesheet with `.node-router`, `.node-switch`, `.node-endpoint` colors.

### Task 4: Link Styling & Legend

**Files:**
- Modify: `webapp/src/components/TopologyGraph.tsx`
- Modify: `webapp/src/components/Legend.tsx` (create if absent)
- Modify: CSS

- [ ] **Step 1:** When drawing links, inspect `link.kind` to choose stroke style: `lldp=solid`, `bridge=dashed`, `gateway=dotted`.

```ts
const strokeDasharray = kind === 'bridge' ? '6,4' : kind === 'gateway' ? '2,4' : 'none'
```

- [ ] **Step 2:** Add tooltip text `kind` + `confidence` on hover.

- [ ] **Step 3:** Create/Update Legend component listing each kind with its stroke sample.

### Task 5: Testing & Verification

**Steps:**
- [ ] **Step 1:** Run `npm --prefix webapp run lint` (if configured) or `npm run test` (if available).
- [ ] **Step 2:** `npm --prefix webapp run build` to ensure bundle succeeds.
- [ ] **Step 3:** `npm --prefix webapp run dev` (manual) to eyeball UI using mock data or live backend.

### Task 6: Documentation

**Files:**
- Modify: `README.md` (UI section)
- Modify: `docs/usage.md` (add note about new UI cues)

- [ ] **Step 1:** Add short paragraph describing node labeling scheme and link legend.
- [ ] **Step 2:** Include screenshot placeholder or instructions on enabling SNMP for richer data.

---
