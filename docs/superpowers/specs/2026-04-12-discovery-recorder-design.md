# 自动化发现落库设计（Discovery Recorder MVP）

## 背景 & 目标
- 目前的 ARP/ICMP/mDNS Runner 只输出日志，没有任何设备/链路持久化逻辑，导致 UI 始终显示“暂无设备”。
- 需要在不大幅重构的前提下，为扫描结果建立“发现 → 去重 → 入库”的最小闭环。
- 目标：当 Runner 观察到设备或链路时，自动写入 SQLite 的 `devices`/`links` 表，并带上 `LastSeen`、接口等基础字段，让前端可以展示拓扑。

## 范围
- **包含**：DiscoveryRecorder 组件、观测事件结构、Runner 调整、数据落库、基本去重与错误日志、单元测试。
- **不包含**：高级协议推理（LLDP 等）、历史统计、前端改版、分布式调度。

## 架构概览
```
Scheduler → Manager → Runner → DiscoveryRecorder → data.Store → SQLite → API → UI
```
- Scheduler 触发扫描，与现有流程一致。
- 每个 Runner 被构造时携带 `Recorder`，只负责产生 `Observation`。
- Recorder 内部维护一个 channel 和 worker：将 Observation 规范化、去重后调用 `data.Store.UpsertDevice/UpsertLink`。
- `store` 仍是单实例，Recorder 只负责协调写入，避免 Runner 直接依赖数据层。

## 关键组件
### DiscoveryRecorder
- 新接口定义：
  ```go
  type Recorder interface {
      RecordDevice(ctx context.Context, obs DeviceObservation)
      RecordLink(ctx context.Context, obs LinkObservation)
      Close()
  }
  ```
- 实现 `recorder`：
  - 缓冲 channel（默认 256）接收 Observation。
  - Worker goroutine 从 channel 读取、整合字段、调用 `data.Store`。
  - 本地 `map[string]time.Time` 做最近一次观测缓存，默认 5s 内重复观测会被丢弃（避免热网段刷爆 DB）。
  - `Close()` 在 main 退出时调用，确保 worker 停止。

### Observation 数据结构
- `DeviceObservation`：
  - `ID`：若有 MAC，取 `strings.ToLower(mac)`；否则 `ip`.
  - `IPv4`, `MAC`, `Hostname`, `Interface`, `Source`（runner 名称）、`Vendor`（可空）、`Confidence`（0-1）。
  - Recorder 若缺少 `ID` 直接丢弃并记录错误。
- `LinkObservation`：
  - `ADevice`, `AInterface`, `BDevice`, `BInterface`, `Media`, `Source`, `Confidence`。
  - `ID` 默认按照 `fmt.Sprintf("%s-%s", ADevice, BDevice)` 排序拼接，保证稳定。

### data.Store 调整
- `UpsertDevice` 已支持 `LastSeen`，Recorder 在写入前确保 `LastSeen=now`。
- `UpsertLink` 同理；新增 `Source/Confidence` 字段如有需要可在未来扩展，目前先用已有 schema。

### Runner 改动
1. **Manager**
   - `NewManager(runners ...Runner)` 改成 `NewManager(recorder Recorder, runners ...Runner)`；内部持有 recorder，并在 `Run` 时将 `Job` + `recorder` 传递。
2. **Runner 接口**
   - 扩展为 `Run(job Job, recorder Recorder) error`，让 Runner 在执行过程中上报观测。
3. **ARP Runner**
   - 读取 `/proc/net/arp` 或通过 `net.Interface` + `syscall` 拉取邻居表（优先使用现有 `hostsFromCIDR` 遍历 + `net.DialUDP`/`arp` 信息）。
   - 对每个主机构造 `DeviceObservation`（包含 `IP`, `MAC`, `Interface`, `Source="arp_nd"`, `Confidence=0.7`）。
4. **ICMP Runner**
   - 在收到 reply 时调用 `recorder.RecordDevice`，ID 以 IP 为准，`Source="icmp"`, `Confidence=0.5`。
5. **Service Runner**
   - mDNS 事件转成 `DeviceObservation`（取 `entry.AddrIPv4[0]`/hostname）。
6. **SNMP Runner**
   - 后续扩展；本轮可先留 TODO。

## 数据流与错误处理
1. Runner 产出 Observation，调用 `Record*`。
2. Recorder 将 obs 写入 channel；如 channel 满，丢弃最旧 obs 并 `log.Warn("discovery backlog", ...)`。
3. Worker：
   - 补全 ID、Interface 等字段。
   - 判断 `lastSeenCache[id]` 与当前时间差，若 <5s 则跳过。
   - 调用 `store.UpsertDevice/UpsertLink`。
   - 遇到 DB 错误记录 `log.Error("record device", "id", id, "err", err)`。

## 测试计划
1. **Recorder 单测**（`internal/scanner/recorder_test.go`）：
   - 使用 stub store 记录收到的对象，验证：
     - channel 写入后最终写 DB；
     - 去重（重复观测不多次写入）；
     - 缓存超时后可以再次写入。
2. **Runner 单测（最少 ICMP/ARP）**：
   - 构造 fake recorder，断言 Runner 在特定条件下会调用 `RecordDevice`。
3. **集成验证**：
   - `go test ./...`。
   - 本地 docker 运行后，通过 `curl /api/v1/devices` 验证至少出现一条记录。

## 部署与迁移
- 仅修改 Go 代码，数据库 schema 无需变动。
- 更新 `cmd/lanmapper/main.go`：创建 recorder（注入 store & logger）→ 传给 Manager → 在 `defer` 中关闭 recorder。
- README 增加“扫描结果持久化”说明，可在后续 PR 处理。

## 风险
- Runner 接口签名变更较大，需要同步调整 `scanner.Manager`、现有测试。
- ARP/ICMP 在大网段下会产生大量观测，需要精心选择缓存窗口与 channel 容量，避免 OOM。
- 在 host 网络模式下，读取 `/proc/net/arp` 必须有相应权限；若容器环境不提供 MAC，将退化为仅 IP。

