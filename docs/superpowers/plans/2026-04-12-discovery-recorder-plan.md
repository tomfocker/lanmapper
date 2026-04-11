# Discovery Recorder Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将扫描 Runner 产生的设备/链路观测写入 SQLite，驱动前端显示真实设备。

**Architecture:** 新增 `DiscoveryRecorder` 负责接收 Runner 的 Observation，通过缓冲 channel 和 worker 去重后调用 `data.Store`。`scanner.Manager` 和 Runner 统一接受 recorder 依赖，Runner 只负责上报事件，Recorder 统一持久化。

**Tech Stack:** Go 1.25、Fiber、SQLite、Go `context`, `time`, `net`.

---

### Task 1: 定义 Observation 与 Recorder 接口

**Files:**
- Create: `internal/scanner/recorder.go`

- [ ] **Step 1:** 在新文件声明数据结构与接口

```go
package scanner

import (
    "context"
    "time"
)

type DeviceObservation struct {
    ID         string
    IPv4       string
    MAC        string
    Hostname   string
    Interface  string
    Source     string
    Vendor     string
    Confidence float64
    ObservedAt time.Time
}

type LinkObservation struct {
    ID         string
    ADevice    string
    AInterface string
    BDevice    string
    BInterface string
    Media      string
    Source     string
    Confidence float64
    ObservedAt time.Time
}

type Recorder interface {
    RecordDevice(context.Context, DeviceObservation)
    RecordLink(context.Context, LinkObservation)
    Close()
}
```

- [ ] **Step 2:** 创建内部实现骨架

```go
type recorder struct {
    store *data.Store
    log   Logger
    devCh chan DeviceObservation
    linkCh chan LinkObservation
    stop chan struct{}
    wg sync.WaitGroup
    cache map[string]time.Time
    cacheTTL time.Duration
}

func NewRecorder(store *data.Store, log Logger) Recorder {
    r := &recorder{
        store: store,
        log: log,
        devCh: make(chan DeviceObservation, 256),
        linkCh: make(chan LinkObservation, 256),
        stop: make(chan struct{}),
        cache: make(map[string]time.Time),
        cacheTTL: 5 * time.Second,
    }
    r.start()
    return r
}
```

### Task 2: 实现 Recorder Worker 与去重

**Files:**
- Modify: `internal/scanner/recorder.go`

- [ ] **Step 1:** 实现 `RecordDevice/RecordLink` 将 observation 写入 channel；channel 满时 `log.Warn` 并丢弃

```go
func (r *recorder) RecordDevice(ctx context.Context, obs DeviceObservation) {
    select {
    case r.devCh <- obs:
    default:
        r.log.Warn("discarding device observation", "id", obs.ID)
    }
}
```

- [ ] **Step 2:** 实现 `start`，开启两个 worker goroutine

```go
func (r *recorder) start() {
    r.wg.Add(2)
    go r.consumeDevices()
    go r.consumeLinks()
}
```

- [ ] **Step 3:** 在 `consumeDevices` 中：
    1. 规范化 ID（`strings.ToLower`）。
    2. 填补 `ObservedAt`。
    3. 判断缓存 `cache[id]`，若 `now.Sub(last) < cacheTTL` 则跳过。
    4. 调用 `store.UpsertDevice`。

```go
func (r *recorder) consumeDevices() {
    defer r.wg.Done()
    for {
        select {
        case <-r.stop:
            return
        case obs := <-r.devCh:
            r.handleDevice(obs)
        }
    }
}
```

- [ ] **Step 4:** `handleDevice` 将 observation 映射到 `models.Device`

```go
func (r *recorder) handleDevice(obs DeviceObservation) {
    if obs.ID == "" && obs.IPv4 == "" {
        r.log.Warn("skip device without id/ip", "source", obs.Source)
        return
    }
    id := obs.ID
    if id == "" {
        id = obs.IPv4
    }
    id = strings.ToLower(id)
    now := time.Now().UTC()
    if obs.ObservedAt.IsZero() {
        obs.ObservedAt = now
    }
    if last, ok := r.cache[id]; ok && now.Sub(last) < r.cacheTTL {
        return
    }
    r.cache[id] = now
    dev := &models.Device{
        ID: id,
        IPv4: obs.IPv4,
        MAC: strings.ToLower(obs.MAC),
        Vendor: obs.Vendor,
        Type: obs.Source,
        LastSeen: obs.ObservedAt,
        Confidence: obs.Confidence,
    }
    if err := r.store.UpsertDevice(context.Background(), dev); err != nil {
        r.log.Error("upsert device", "id", id, "err", err)
    }
}
```

- [ ] **Step 5:** `consumeLinks` 同理，将 observation 转为 `models.Link`

- [ ] **Step 6:** 实现 `Close`：关闭 `stop`、等待 `wg`

```go
func (r *recorder) Close() {
    close(r.stop)
    r.wg.Wait()
}
```

### Task 3: 更新 Runner/Manager 接口

**Files:**
- Modify: `internal/scanner/types.go`
- Modify: `internal/scanner/manager.go`
- Modify: `internal/scanner/manager_test.go`

- [ ] **Step 1:** 修改 `Runner` 接口

```go
type Runner interface {
    Name() string
    Run(job Job, recorder Recorder) error
}
```

- [ ] **Step 2:** `Manager` 持有 `Recorder`

```go
type Manager struct {
    runners []Runner
    recorder Recorder
    ...
}

func NewManager(recorder Recorder, runners ...Runner) *Manager { ... }
```

- [ ] **Step 3:** 在 `Start` 的 worker 中调用 `r.Run(job, m.recorder)`

- [ ] **Step 4:** 更新 `manager_test.go`，使用 fake recorder 断言 `Run` 被调用

```go
type stubRecorder struct{}
func (s *stubRecorder) RecordDevice(context.Context, DeviceObservation) {}
...
```

### Task 4: Runner 上报 Observation

**Files:**
- Modify: `internal/scanner/protocol_arpnd.go`
- Modify: `internal/scanner/protocol_icmp.go`
- Modify: `internal/scanner/protocol_services.go`

- [ ] **Step 1:** 在 ARP Runner 中解析 `/proc/net/arp`（使用 `net.ParseIP` + `strings.Fields`），为每个条目调用 `recorder.RecordDevice`

```go
recorder.RecordDevice(ctx, DeviceObservation{
    ID: mac,
    IPv4: ip,
    MAC: mac,
    Interface: iface.Name,
    Source: r.Name(),
    Confidence: 0.7,
})
```

- [ ] **Step 2:** ICMP Runner 在收到 reply 时上报

```go
r.log.Info("icmp reply", ...)
recorder.RecordDevice(ctx, DeviceObservation{
    ID: ip.String(),
    IPv4: ip.String(),
    Source: r.Name(),
    Confidence: 0.5,
})
```

- [ ] **Step 3:** Service Runner 在 mDNS callback 中上报

```go
recorder.RecordDevice(context.Background(), DeviceObservation{
    ID: entry.Instance,
    IPv4: entry.AddrIPv4[0].String(),
    Hostname: entry.HostName,
    Source: r.Name(),
    Confidence: 0.3,
})
```

### Task 5: 新增 Recorder 与 Runner 测试

**Files:**
- Create: `internal/scanner/recorder_test.go`
- Modify: `internal/scanner/protocol_icmp_test.go`（若不存在则新建）

- [ ] **Step 1:** `recorder_test.go` 创建 stub store

```go
type stubStore struct { devices []*models.Device }
func (s *stubStore) UpsertDevice(ctx context.Context, d *models.Device) error {
    s.devices = append(s.devices, d)
    return nil
}
```

- [ ] **Step 2:** 测试 `RecordDevice` 会写入一次，重复观测在 TTL 内不重复

```go
r := NewRecorder(store, loggerLForTest()).(*recorder)
r.RecordDevice(context.Background(), DeviceObservation{ID: "mac", IPv4: "192.168.1.2"})
time.Sleep(10 * time.Millisecond)
require.Len(t, store.devices, 1)
```

- [ ] **Step 3:** 测试 `Close` 能退出 goroutine（使用 `time.NewTimer` + `Close`）

- [ ] **Step 4:** 为 ICMP Runner 编写测试：fake recorder 记录调用次数

```go
type fakeRecorder struct { count int }
func (f *fakeRecorder) RecordDevice(ctx context.Context, obs DeviceObservation) { f.count++ }
```

- [ ] **Step 5:** `go test ./internal/scanner -run TestRecorder`

### Task 6: 主程序接入 Recorder

**Files:**
- Modify: `cmd/lanmapper/main.go`

- [ ] **Step 1:** 创建 recorder

```go
rec := scanner.NewRecorder(store, log)
defer rec.Close()
```

- [ ] **Step 2:** 初始化 Manager 时传入 recorder

```go
mgr := scanner.NewManager(rec, runners...)
```

- [ ] **Step 3:** 运行 `go test ./cmd/lanmapper ./internal/scanner`

### Task 7: 文档与最终验证

**Files/Commands:**
- Modify: `README.md`
- Modify: `docs/usage.md`
- Run: `go test ./...`
- (Optional) `docker build -t lanmapper .`

- [ ] **Step 1:** README 增加一段“默认扫描会自动落库，UI 会显示设备列表”的说明

- [ ] **Step 2:** `docs/usage.md` 增加 `curl /api/v1/devices` 返回示例

- [ ] **Step 3:** 执行 `go test ./...`

- [ ] **Step 4:** 可选构建 Docker 镜像验证

- [ ] **Step 5:** `git status` 确认改动，准备下一阶段提交

---
