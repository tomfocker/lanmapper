# Autoscan & Scheduler Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 自动识别容器所在默认网段、主动调度扫描任务并提供 REST 触发能力，让部署后无需配置即可发现设备。

**Architecture:** 新增网段检测模块读取路由/接口信息，生成 CIDR 列表；在主程序启动与 API 中调用调度器入队扫描任务，同时把扫描元数据写入 SQLite；可选周期任务依据 `scan_interval` 触发。

**Tech Stack:** Go 1.25（net、os、time、database/sql）、Fiber API、SQLite（modernc.org/sqlite）、React 前端无需变动。

---

### Task 1: 网段检测模块

**Files:**
- Create: `internal/scanner/interfaces.go`
- Create: `internal/scanner/interfaces_test.go`

- [ ] **Step 1:** 在 `internal/scanner/interfaces.go` 实现接口/路由枚举

```go
package scanner

import (
    "bufio"
    "net"
    "os"
    "strings"
)

type DetectedCIDR struct {
    CIDR      *net.IPNet
    Interface string
}

func DetectDefaultCIDRs() ([]DetectedCIDR, error) {
    ifaceName, err := defaultRouteIface("/proc/net/route")
    if err != nil {
        ifaceName = firstCandidateIface()
    }
    if ifaceName == "" {
        return nil, fmt.Errorf("no active interface found")
    }
    iface, err := net.InterfaceByName(ifaceName)
    if err != nil {
        return nil, err
    }
    var cidrs []DetectedCIDR
    addrs, _ := iface.Addrs()
    for _, addr := range addrs {
        if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
            cidrs = append(cidrs, DetectedCIDR{CIDR: canonicalize(ipnet), Interface: iface.Name})
        }
    }
    return dedupeCIDRs(cidrs), nil
}
```

- [ ] **Step 2:** 在同文件实现 `defaultRouteIface`, `firstCandidateIface`, `dedupeCIDRs`, `canonicalize` 等 helper（分别解析 `/proc/net/route`、fallback 至第一个非 loopback up 接口、对 CIDR 排序去重、对 /32 推断 /24）。

- [ ] **Step 3:** 添加单元测试 `internal/scanner/interfaces_test.go`

```go
func TestDefaultRouteIface(t *testing.T) {
    tmp := t.TempDir()
    content := "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\tMTU\tWindow\tIRTT\n" +
        "eth0\t00000000\t0102A8C0\t0003\t0\t0\t0\t00FFFFFF\t0\t0\t0\n"
    if err := os.WriteFile(filepath.Join(tmp, "route"), []byte(content), 0644); err != nil { t.Fatal(err) }
    iface, err := defaultRouteIface(filepath.Join(tmp, "route"))
    require.NoError(t, err)
    require.Equal(t, "eth0", iface)
}
```

- [ ] **Step 4:** 运行 `go test ./internal/scanner -run TestDefaultRouteIface` 预期通过。

### Task 2: 扫描调度器与主程序整合

**Files:**
- Modify: `cmd/lanmapper/main.go`
- Create: `internal/scanner/scheduler.go`

- [ ] **Step 1:** 新建 `internal/scanner/scheduler.go`

```go
type Scheduler struct {
    mgr      *Manager
    store    *data.Store
    ticker   *time.Ticker
    stopChan chan struct{}
}

func NewScheduler(mgr *Manager, store *data.Store) *Scheduler { ... }

func (s *Scheduler) Trigger(ctx context.Context, cidrs []DetectedCIDR) (string, error) {
    scanID := uuid.NewString()
    if err := s.store.InsertScan(ctx, scanID, cidrs); err != nil { return "", err }
    for _, target := range cidrs {
        job := Job{CIDR: target.CIDR, Interface: target.Interface, ScanID: scanID}
        s.mgr.Enqueue(job)
    }
    return scanID, nil
}

func (s *Scheduler) StartInterval(ctx context.Context, cidrs []DetectedCIDR, interval time.Duration) {
    if interval <= 0 { return }
    s.ticker = time.NewTicker(interval)
    go func() {
        for {
            select {
            case <-ctx.Done():
                s.ticker.Stop()
                return
            case <-s.ticker.C:
                s.Trigger(ctx, cidrs)
            }
        }
    }()
}
```

- [ ] **Step 2:** 调整 `cmd/lanmapper/main.go`

```go
detected, err := scanner.DetectDefaultCIDRs()
targets := mergeCIDRs(detected, cfg.ScanCIDR)
sched := scanner.NewScheduler(mgr, store)
if _, err := sched.Trigger(ctx, targets); err != nil { log.Error("initial scan", "err", err) }
sched.StartInterval(ctx, targets, cfg.ScanInterval)
```

`mergeCIDRs` 将字符串 CIDR 转 `net.IPNet` 并覆盖 `Interface`（配置显式指定时 Interface 可空）。

- [ ] **Step 3:** 确保在 `main.go` 中 `mgr.Start(ctx)` 之后调用 `sched`，同时在 `defer` 中停止 ticker（如果实现了 stop）。

- [ ] **Step 4:** 运行 `go test ./cmd/lanmapper ./internal/scanner`。

### Task 3: API 触发与验证

**Files:**
- Modify: `internal/api/routes.go`
- Modify: `internal/api/server.go`

- [ ] **Step 1:** `server.Start` 接收 `sched *scanner.Scheduler`，Fiber `App` 的 `/api/v1/scans` 使用 scheduler。

- [ ] **Step 2:** 更新路由逻辑，解析可选 payload 并触发：

```go
type scanRequest struct {
    CIDR      []string `json:"cidr"`
    Interface string   `json:"interface"`
}
```

- [ ] **Step 3:** 添加 `schedulerTargets` helper + 测试，校验非法 CIDR 返回错误。

- [ ] **Step 4:** 本地 `curl -X POST http://localhost:8080/api/v1/scans`，确认 `scan_id` 返回。

### Task 4: `scans` 数据存取

**Files:**
- Modify: `internal/data/store.go`
- Modify: `internal/data/migrations.go`
- Create: `internal/models/scan.go`

- [ ] **Step 1:** 定义 `models.Scan`，字段含 `ID, Status, StartedAt, FinishedAt, Targets`。

- [ ] **Step 2:** 如 `migrations.go` 尚无 `targets/status` 列，追加迁移。

- [ ] **Step 3:** `store.go` 新增 `InsertScan`, `FinishScan`，使用 JSON 保存 targets。

- [ ] **Step 4:** 编写测试：使用 `t.TempDir()` 创建数据库，调用插入/更新，查询验证。

### Task 5: 文档与验证

**Files/Commands:**
- Modify: `README.md`
- Modify: `docs/usage.md`
- Run: `go test ./...`
- (Optional) `docker build -t lanmapper .`

- [ ] **Step 1:** README 增加“默认自动探测网段，SCAN_INTERVAL/SCAN_CIDR 用法”。

- [ ] **Step 2:** docs/usage.md 描述 `/api/v1/scans` 请求示例。

- [ ] **Step 3:** `go test ./...`，保证所有新增测试通过。

- [ ] **Step 4:** 可选 `docker build -t lanmapper .`，验证镜像可构建。

---

## 自检
- 覆盖自动探测、调度器、API、数据持久化、文档。
- 所有步骤提供具体文件路径和代码片段，无占位描述。
- 交付后具备可验证路径：`go test`、`curl /api/v1/scans`、Docker 构建。
