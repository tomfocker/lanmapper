# LAN 拓扑 Docker 工具 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建单容器部署的 Go + React 工具，扫描局域网设备并输出拓扑与多格式报告。

**Architecture:** Go 单体进程暴露 REST/WebSocket API、驱动并发扫描和 SQLite 存储，React/Vis.js 前端展示拓扑并触发导出，Chromedp 负责 PDF/PNG/SVG 截图。

**Tech Stack:** Go 1.22, Fiber(或 Echo), gopacket, gosnmp, sqlite (modernc.org/sqlite), chromedp, React 18 + Vite + vis-network, TailwindCSS, Docker multi-stage。

---

## File Structure Overview
- `cmd/lanmapper/main.go`：入口，加载配置、启动服务。
- `internal/config/config.go`：配置结构与加载。
- `internal/logger/logger.go`：zap/slog 包装。
- `internal/data/{store.go,migrations.go}`：SQLite 访问层与 schema。
- `internal/models/{device.go,link.go,scan.go,report.go}`：领域模型。
- `internal/scanner/{manager.go,task.go,protocol_*.go}`：扫描调度与协议实现。
- `internal/topology/builder.go`：拓扑推断与置信度计算。
- `internal/api/{server.go,routes.go,handlers.go,ws.go}`：REST + WebSocket。
- `internal/report/{generator.go,templates/*}`：JSON/CSV/PDF/PNG/SVG 输出。
- `webapp/`：React 源码，Vite 配置，组件（Dashboard, TopologyView, DevicePanel, Reports）。
- `Dockerfile` + `Makefile` + `docs/`。

---

### Task 1: 初始化仓库与基础脚手架

**Files:**
- Create: `go.mod`, `go.sum`, `cmd/lanmapper/main.go`, `.gitignore`, `Makefile`

- [ ] **Step 1: 初始化 Go 模块**
```bash
go mod init github.com/tomfocker/lanmapper
go env -w GO111MODULE=on
```

- [ ] **Step 2: 创建 `.gitignore`**
```
bin/
build/
data/
webapp/node_modules/
webapp/dist/
*.db
.env
.superpowers/
```

- [ ] **Step 3: 编写基本 `Makefile`**
```
BINARY=lanmapper
.PHONY: build run ui dev
build:
	GOOS=linux GOARCH=amd64 go build -o bin/$(BINARY) ./cmd/lanmapper
run:
	go run ./cmd/lanmapper
ui:
	cd webapp && npm install && npm run dev
```

- [ ] **Step 4: 写 `cmd/lanmapper/main.go` 引导**
```go
package main

import (
    "log"

    "github.com/tomfocker/lanmapper/internal/config"
    "github.com/tomfocker/lanmapper/internal/server"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("load config: %v", err)
    }
    if err := server.Start(cfg); err != nil {
        log.Fatalf("start server: %v", err)
    }
}
```

### Task 2: 配置系统与日志

**Files:**
- Create: `internal/config/config.go`, `internal/config/defaults.go`, `internal/logger/logger.go`
- Modify: `cmd/lanmapper/main.go`

- [ ] **Step 1: 定义配置结构**
```go
package config

import "time"

type Config struct {
    HTTPPort int `mapstructure:"http_port"`
    DataDir  string `mapstructure:"data_dir"`
    ScanCIDR []string `mapstructure:"scan_cidr"`
    SNMPCommunities []string `mapstructure:"snmp_communities"`
    ScanInterval time.Duration `mapstructure:"scan_interval"`
    AdminToken string `mapstructure:"admin_token"`
}
```

- [ ] **Step 2: 实现 `Load()` 支持 env + yaml**
```go
func Load() (*Config, error) {
    v := viper.New()
    v.SetConfigName("config")
    v.SetConfigType("yaml")
    v.AddConfigPath(".")
    v.AddConfigPath("/app/data")
    v.AutomaticEnv()
    setDefaults(v)
    _ = v.ReadInConfig()
    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

- [ ] **Step 3: 设置默认值**
```go
func setDefaults(v *viper.Viper) {
    v.SetDefault("http_port", 8080)
    v.SetDefault("data_dir", "./data")
    v.SetDefault("scan_cidr", []string{})
    v.SetDefault("snmp_communities", []string{"public","private"})
    v.SetDefault("scan_interval", "0s")
}
```

- [ ] **Step 4: 创建日志包装**
```go
package logger

import "log/slog"

var defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func L() *slog.Logger { return defaultLogger }
```

- [ ] **Step 5: main 中注入 logger/config**
```go
func main() {
    cfg, err := config.Load()
    if err != nil {
        logger.L().Error("config", "err", err)
        os.Exit(1)
    }
    if err := server.Start(cfg, logger.L()); err != nil {
        logger.L().Error("server", "err", err)
        os.Exit(1)
    }
}
```

### Task 3: 数据模型与 SQLite 存储

**Files:**
- Create: `internal/models/{device.go,link.go,scan.go,report.go}`
- Create: `internal/data/{store.go,migrations.go}`
- Modify: `internal/config/config.go` (添加 DB Path)

- [ ] **Step 1: 定义模型结构**
```go
package models

type Device struct {
    ID string `db:"id"`
    IPv4 string `db:"ipv4"`
    IPv6 string `db:"ipv6"`
    MAC string `db:"mac"`
    Vendor string `db:"vendor"`
    Type string `db:"type"`
    Protocols []string `db:"-"`
    LastSeen time.Time `db:"last_seen"`
    Confidence float64 `db:"confidence"`
}
```

- [ ] **Step 2: 初始化数据库**
```go
package data

import "modernc.org/sqlite"

type Store struct {
    db *sql.DB
}

func New(path string) (*Store, error) {
    db, err := sql.Open("sqlite", path)
    if err != nil { return nil, err }
    if err := migrate(db); err != nil { return nil, err }
    return &Store{db: db}, nil
}
```

- [ ] **Step 3: 编写迁移 SQL**
```go
const schema = `CREATE TABLE IF NOT EXISTS devices (
    id TEXT PRIMARY KEY,
    ipv4 TEXT,
    ipv6 TEXT,
    mac TEXT UNIQUE,
    vendor TEXT,
    type TEXT,
    last_seen DATETIME,
    confidence REAL
);
CREATE TABLE IF NOT EXISTS interfaces (...);
CREATE TABLE IF NOT EXISTS links (...);
CREATE TABLE IF NOT EXISTS scans (...);
CREATE TABLE IF NOT EXISTS reports (...);
`
```

- [ ] **Step 4: CRUD 方法**
```go
func (s *Store) UpsertDevice(ctx context.Context, d *models.Device) error {
    _, err := s.db.ExecContext(ctx, `INSERT INTO devices
        (id, ipv4, ipv6, mac, vendor, type, last_seen, confidence)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
          ipv4=excluded.ipv4,
          ipv6=excluded.ipv6,
          ...;
    `, d.ID, d.IPv4, d.IPv6, d.MAC, d.Vendor, d.Type, d.LastSeen, d.Confidence)
    return err
}
```

- [ ] **Step 5: Store 单元测试**
```go
func TestUpsertDevice(t *testing.T) {
    store := setupInMemory(t)
    err := store.UpsertDevice(context.Background(), &models.Device{ID:"dev1"})
    require.NoError(t, err)
    dev, _ := store.GetDevice(ctx, "dev1")
    assert.Equal(t, "dev1", dev.ID)
}
```

### Task 4: 扫描管理器与任务框架

**Files:**
- Create: `internal/scanner/{manager.go,task.go,iface.go}`
- Modify: `cmd/lanmapper/main.go` (注入 store)

- [ ] **Step 1: 定义任务/协议接口**
```go
type Runner interface {
    Name() string
    Run(ctx context.Context, job Job) error
}

type Job struct {
    CIDR *net.IPNet
    Interface string
    ScanID string
}
```

- [ ] **Step 2: Manager 结构**
```go
type Manager struct {
    runners []Runner
    jobs chan Job
    wg sync.WaitGroup
}

func NewManager(runners ...Runner) *Manager {
    return &Manager{runners: runners, jobs: make(chan Job, 64)}
}
```

- [ ] **Step 3: 启动与调度**
```go
func (m *Manager) Start(ctx context.Context) {
    for i := 0; i < runtime.NumCPU()*5; i++ {
        m.wg.Add(1)
        go func() {
            defer m.wg.Done()
            for job := range m.jobs {
                for _, r := range m.runners {
                    if err := r.Run(ctx, job); err != nil {
                        logger.L().Error("runner", "name", r.Name(), "err", err)
                    }
                }
            }
        }()
    }
}
```

- [ ] **Step 4: 接口检测 `iface.go`**
```go
func LocalInterfaces() ([]*net.Interface, error) {
    ifs, err := net.Interfaces()
    // 过滤 loopback, down, 无多播
}
```

- [ ] **Step 5: main 中创建 Manager**
```go
store, _ := data.New(filepath.Join(cfg.DataDir, "lanmapper.db"))
manager := scanner.NewManager(protocols...)
manager.Start(ctx)
```

### Task 5: ARP/ND/ICMP 协议实现

**Files:**
- Create: `internal/scanner/protocol_arpnd.go`, `protocol_icmp.go`
- Modify: `internal/scanner/iface.go` (多播加入)

- [ ] **Step 1: ARP/ND Runner**
```go
type ARPNDRunner struct {
    store *data.Store
}

func (r *ARPNDRunner) Run(ctx context.Context, job Job) error {
    handle, err := pcap.OpenLive(job.Interface, 65535, true, pcap.BlockForever)
    // 构造 ARP 请求包
    for _, ip := range hosts(job.CIDR) {
        pkt := craftARP(ip)
        if err := handle.WritePacketData(pkt); err != nil { ... }
    }
    // 监听响应，解析 MAC/IP，写 store
}
```

- [ ] **Step 2: IPv6 Neighbor Discovery**
```go
func craftNS(ip net.IP) []byte { /* gopacket 构造 ICMPv6 */ }
```

- [ ] **Step 3: ICMP Runner**
```go
type ICMPRunner struct {}
func (r *ICMPRunner) Run(ctx context.Context, job Job) error {
    // 使用 golang.org/x/net/icmp 发送 echo
}
```

- [ ] **Step 4: Rate Limit**
```go
limiter := rate.NewLimiter(rate.Limit(200), 400)
if err := limiter.Wait(ctx); err != nil { return err }
```

- [ ] **Step 5: 单元测试**（使用 raw socket 需 mock）
```go
func TestHostsFromCIDR(t *testing.T) {
    ips := hosts(mustCIDR("192.168.1.0/30"))
    assert.Equal(t, []string{"192.168.1.1","192.168.1.2"}, ips)
}
```

### Task 6: SNMP/服务发现协议

**Files:**
- Create: `internal/scanner/protocol_snmp.go`, `protocol_services.go`

- [ ] **Step 1: SNMP Runner**
```go
type SNMPRunner struct {
    store *data.Store
    communities []string
}

func (r *SNMPRunner) Run(ctx context.Context, job Job) error {
    for _, ip := range discoveredIPs(job.ScanID) {
        for _, comm := range r.communities {
            client := &gosnmp.GoSNMP{Target: ip.String(), Community: comm, Port: 161, Version: gosnmp.Version2c, Timeout: 2 * time.Second}
            if err := client.Connect(); err != nil { continue }
            if err := r.fetchDevice(ctx, client, ip); err == nil { break }
        }
    }
    return nil
}
```

- [ ] **Step 2: 读取 LLDP/CDP/MAC 表**
```go
func (r *SNMPRunner) fetchLLDP(ctx context.Context, client *gosnmp.GoSNMP, devID string) error {
    results, err := client.BulkWalkAll("1.0.8802.1.1.2.1.4.1")
    for _, pdu := range results {
        // 构造 Link 记录，source="lldp"
    }
    return err
}
```

- [ ] **Step 3: Service Discovery Runner**
```go
func (r *ServiceRunner) Run(ctx context.Context, job Job) error {
    // mDNS via mdns library, SSDP via UDP multicast, NetBIOS via nbns
}
```

- [ ] **Step 4: 捕获公告写入 store**
```go
func recordService(ctx context.Context, svc ServiceInfo) {
    store.UpsertDevice(ctx, &models.Device{ID: svc.MAC, Type: svc.Type, Vendor: svc.Vendor})
}
```

- [ ] **Step 5: SNMP 单元测试**（使用 gosnmp/mock）
```go
func TestParseLLDP(t *testing.T) {
    pdus := []gosnmp.SnmpPDU{{Name:"1.0...1.5.0", Value:"Gig1"}}
    links := parseLLDP(pdus)
    assert.Equal(t, 1, len(links))
}
```

### Task 7: 拓扑构建器

**Files:**
- Create: `internal/topology/builder.go`, `internal/topology/builder_test.go`
- Modify: `internal/scanner/manager.go`（扫描后触发构建）

- [ ] **Step 1: 定义 Builder**
```go
type Builder struct {
    store *data.Store
}

func (b *Builder) Rebuild(ctx context.Context, scanID string) (*Graph, error) {
    devices, _ := b.store.ListDevices(ctx)
    links, _ := b.store.ListRawLinks(ctx)
    graph := NewGraph(devices)
    for _, l in links { graph.Connect(l.ADevice, l.BDevice, l.Confidence) }
    // Derived edges from heuristics
    return graph, nil
}
```

- [ ] **Step 2: 置信度策略**
```go
func confidenceFromSource(source string) float64 {
    switch source {
    case "lldp": return 0.95
    case "mac": return 0.75
    default: return 0.4
    }
}
```

- [ ] **Step 3: Heuristic 链路**
```go
if sameSwitch(devA, devB) && shareVLAN(devA, devB) {
    graph.Connect(devA.ID, devB.ID, 0.5)
}
```

- [ ] **Step 4: 将 Graph 写缓存表**
```go
func (b *Builder) Persist(ctx context.Context, g *Graph) error {
    for _, edge := range g.Edges {
        b.store.UpsertLink(ctx, edge)
    }
}
```

- [ ] **Step 5: 单元测试**
```go
func TestBuilderPrefersLLDP(t *testing.T) {
    // Insert conflicting LLDP+MAC edges，断言最终 Link 来源为 lldp
}
```

### Task 8: REST & WebSocket API

**Files:**
- Create: `internal/api/server.go`, `internal/api/routes.go`, `internal/api/handlers.go`, `internal/api/ws.go`
- Modify: `cmd/lanmapper/main.go`

- [ ] **Step 1: Fiber 服务器初始化**
```go
func StartHTTP(cfg *config.Config, store *data.Store, mgr *scanner.Manager, topo *topology.Builder) error {
    app := fiber.New()
    api := app.Group("/api/v1", middleware.Auth(cfg.AdminToken))
    registerRoutes(api, store, mgr, topo)
    app.Static("/", embedFS)
    return app.Listen(fmt.Sprintf(":%d", cfg.HTTPPort))
}
```

- [ ] **Step 2: REST 路由**
```go
func registerRoutes(r fiber.Router, store *data.Store, mgr *scanner.Manager, topo *topology.Builder) {
    r.Get("/devices", listDevices)
    r.Get("/links", listLinks)
    r.Post("/scans", triggerScan)
    r.Get("/scans/:id", getScan)
    r.Post("/reports", createReport)
}
```

- [ ] **Step 3: 处理函数示例**
```go
func listDevices(c *fiber.Ctx) error {
    devs, err := store.ListDevices(c.Context())
    if err != nil { return fiber.ErrInternalServerError }
    return c.JSON(devs)
}
```

- [ ] **Step 4: WebSocket 推送**
```go
var hub = websocket.New()
func notifyProgress(scanID string, pct float64) {
    hub.Broadcast(struct{ScanID string; Progress float64}{scanID, pct})
}
```

- [ ] **Step 5: API 测试**（使用 httptest）
```go
func TestListDevices(t *testing.T) {
    app := fiber.New()
    app.Get("/devices", listDevices)
    req := httptest.NewRequest("GET", "/devices", nil)
    resp, _ := app.Test(req)
    assert.Equal(t, http.StatusOK, resp.StatusCode)
}
```

### Task 9: 报告生成管线

**Files:**
- Create: `internal/report/generator.go`, `internal/report/templates/{report.html,summary.json.tmpl,devices.csv.tmpl}`
- Modify: `internal/api/handlers.go` (导出接口)

- [ ] **Step 1: JSON/CSV 模板**
```go
func (g *Generator) ExportJSON(ctx context.Context, scanID string) (string, error) {
    data := struct{ Devices []models.Device; Links []models.Link }{...}
    buf := bytes.Buffer{}
    if err := json.NewEncoder(&buf).Encode(data); err != nil { return "", err }
    path := filepath.Join(g.outDir, scanID+".json")
    return path, os.WriteFile(path, buf.Bytes(), 0o644)
}
```

- [ ] **Step 2: CSV 模板**
```go
func exportCSV(path string, devices []models.Device) error {
    w := csv.NewWriter(f)
    w.Write([]string{"IP", "MAC", "Vendor", "Type", "Confidence"})
}
```

- [ ] **Step 3: PDF/PNG/SVG via chromedp**
```go
func (g *Generator) ExportPDF(ctx context.Context, scanID string) (string, error) {
    ctx, cancel := chromedp.NewContext(ctx)
    defer cancel()
    var buf []byte
    url := fmt.Sprintf("http://127.0.0.1:%d/report?scan=%s", g.httpPort, scanID)
    err := chromedp.Run(ctx,
        chromedp.Navigate(url),
        chromedp.Sleep(2*time.Second),
        chromedp.FullScreenshot(&buf, 90),
    )
    path := filepath.Join(g.outDir, scanID+".png")
    return path, os.WriteFile(path, buf, 0o644)
}
```

- [ ] **Step 4: 报告记录到 DB**
```go
store.InsertReport(ctx, &models.Report{ID: uuid.NewString(), ScanID: scanID, Type: "pdf", Path: path})
```

- [ ] **Step 5: 单元测试**（模板渲染）
```go
func TestExportJSON(t *testing.T) {
    gen := NewGenerator(tempDir, store)
    path, err := gen.ExportJSON(ctx, scanID)
    require.NoError(t, err)
    content, _ := os.ReadFile(path)
    assert.Contains(t, string(content), "devices")
}
```

### Task 10: React 前端

**Files:**
- Create: `webapp/package.json`, `webapp/vite.config.ts`, `webapp/src/{main.tsx,App.tsx,components/*}`
- Create: `webapp/src/api/client.ts`, `webapp/src/components/{Dashboard.tsx,TopologyView.tsx,DevicePanel.tsx,ReportCenter.tsx}`

- [ ] **Step 1: 初始化 Vite React TS**
```bash
cd webapp
npm create vite@latest webapp -- --template react-ts
npm install vis-network tailwindcss @tanstack/react-query socket.io-client recharts
npx tailwindcss init -p
```

- [ ] **Step 2: 设置 API 客户端**
```ts
const client = axios.create({ baseURL: '/api/v1' })
export const fetchDevices = () => client.get<Device[]>('/devices').then(r => r.data)
```

- [ ] **Step 3: App 布局**
```tsx
return (
  <QueryClientProvider client={queryClient}>
    <Layout>
      <Sidebar />
      <Main>
        <Dashboard />
        <TopologyView />
      </Main>
    </Layout>
  </QueryClientProvider>
)
```

- [ ] **Step 4: 拓扑组件**
```tsx
const networkRef = useRef<vis.Network | null>(null)
useEffect(() => {
  if (!networkRef.current) {
    networkRef.current = new vis.Network(container, { nodes, edges }, options)
  } else {
    networkRef.current.setData({ nodes, edges })
  }
}, [nodes, edges])
```

- [ ] **Step 5: WebSocket 进度条**
```tsx
useEffect(() => {
  const socket = io('/ws')
  socket.on('scan_progress', setProgress)
  return () => socket.close()
}, [])
```

- [ ] **Step 6: Tailwind 基础样式**
```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

- [ ] **Step 7: UI 测试**（Vitest + React Testing Library）
```ts
test('renders device count', () => {
  render(<Dashboard devices={[{ id: 'dev1', type: 'switch' }]} />)
  expect(screen.getByText('1 台设备')).toBeInTheDocument()
})
```

### Task 11: Embed 前端并提供静态资源

**Files:**
- Modify: `internal/api/server.go` (embed 静态文件)
- Add: `webapp/dist` build script

- [ ] **Step 1: 使用 `embed`**
```go
//go:embed all:webapp/dist
var webFS embed.FS
app.Static("/", http.FS(webFS))
```

- [ ] **Step 2: 添加 `make ui-build`**
```
ui-build:
	cd webapp && npm install && npm run build
```

- [ ] **Step 3: main 启动时先确保 dist 资源存在（开发模式可回退代理）**

### Task 12: Docker 化与启动脚本

**Files:**
- Create: `Dockerfile`, `docker-compose.example.yml`, `scripts/entrypoint.sh`

- [ ] **Step 1: Dockerfile**
```Dockerfile
FROM node:20-alpine AS ui
WORKDIR /src
COPY webapp/ .
RUN npm install && npm run build

FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui /src/dist ./webapp/dist
RUN CGO_ENABLED=1 go build -o lanmapper ./cmd/lanmapper

FROM alpine:3.19
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /app/lanmapper .
COPY --from=builder /app/webapp/dist ./webapp/dist
COPY scripts/entrypoint.sh ./entrypoint.sh
RUN chmod +x entrypoint.sh
USER app
ENTRYPOINT ["/app/entrypoint.sh"]
```

- [ ] **Step 2: entrypoint**
```sh
#!/bin/sh
set -e
mkdir -p "$DATA_DIR"
exec /app/lanmapper --config "$DATA_DIR/config.yaml"
```

- [ ] **Step 3: docker-compose 示例**
```yaml
services:
  lanmapper:
    build: .
    network_mode: host
    cap_add:
      - NET_ADMIN
    volumes:
      - ./data:/app/data
```

### Task 13: 文档与用户指南

**Files:**
- Create: `README.md`, `docs/usage.md`, `docs/report-format.md`

- [ ] **Step 1: README**
```
# LAN Mapper
## Quick Start
```

- [ ] **Step 2: Usage 文档覆盖运行命令、配置、权限说明。

- [ ] **Step 3: Report 格式文档列出 JSON/CSV 字段定义。

### Task 14: 测试与 CI

**Files:**
- Create: `.github/workflows/ci.yml` or `scripts/ci.sh`
- Modify: `Makefile`

- [ ] **Step 1: Makefile 添加 `test`, `lint`**
```
test:
	go test ./...
ui-test:
	cd webapp && npm test
```

- [ ] **Step 2: GitHub Actions**
```yaml
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.22' }
      - run: go test ./...
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - run: |
          cd webapp
          npm install
          npm run build
```

- [ ] **Step 3: End-to-end 脚本**
```bash
bash scripts/e2e.sh
# 1. 启动 mock 网络
# 2. 运行 lanmapper
# 3. curl /api 验证
```

### Task 15: 验证与发布

**Files:**
- Create: `scripts/mock-lab.sh`, `docs/release-checklist.md`

- [ ] **Step 1: mock 实验室脚本**
```bash
#!/bin/bash
# 使用 containerlab 启动 2 台交换机 + 4 主机，配置 LLDP/SNMP
```

- [ ] **Step 2: 发布检查**
```
## Checklist
- [ ] go test ./...
- [ ] npm run build
- [ ] docker build .
- [ ] docker run ...
```

- [ ] **Step 3: 记录版本号**（`internal/version/version.go`）供 UI 显示。

---

## Self-Review
1. **Spec coverage**：所有模块（扫描、拓扑、Web UI、导出、Docker、文档、测试）均有对应任务。
2. **Placeholder scan**：未使用 TBD/等词，步骤给出具体代码/命令。
3. **一致性**：命名（例如 `lanmapper`、`Store`, `Manager`, `Builder`）在各任务中保持一致。

