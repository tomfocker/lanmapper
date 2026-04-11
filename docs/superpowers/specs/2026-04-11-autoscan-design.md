# 自动识别网段与扫描调度设计

## 背景
- 目前镜像默认 `scan_cidr` 为空，用户若不配置无法扫描任何设备。
- API `/api/v1/scans` 仅返回字符串，没有实际触发扫描；Manager 也没有任务入队。
- 希望容器部署后即刻基于自身默认网段开始扫描，并支持周期/手动触发。

## 目标
1. **网段自动发现**：启动时解析系统路由和接口，选默认路由接口生成 CIDR 列表；作为默认扫描目标。
2. **调度器**：新增 Scheduler 统一调度扫描；支持一次性启动与 `scan_interval` 周期任务。
3. **API 触发**：`POST /api/v1/scans` 可选携带 CIDR，立即触发新的扫描并返回 `scan_id`。
4. **数据记录**：`scans` 表记录每次扫描状态、目标、时间戳，为后续历史/报告做准备。
5. **日志与提示**：自动识别失败或无可用网段时必须输出 warning，指导用户手动配置。

## 非目标
- 多接口并行策略（暂以默认路由接口为主）。
- UI 上的“立即扫描”按钮（留后续接入 API）。
- 新的拓扑推理/协议逻辑。

## 方案
1. **网段检测**
   - 新模块 `internal/scanner/interfaces.go`：
     - 解析 `/proc/net/route` 找到 `Destination == 0` 的接口；失败时 fallback 至第一个非 loopback、UP 的接口。
     - 读取接口地址列表，生成 `DetectedCIDR{CIDR *net.IPNet, Interface string}`。
     - 对 `/32` 这样的点对点掩码可降级到 `/24`（或保留原样且提示）。
     - 去重/排序，返回 slice。

2. **调度器**
   - 新建 `internal/scanner/scheduler.go`，持有 `*scanner.Manager` 和 `*data.Store`。
   - 方法 `Trigger(ctx, targets)`：生成 `scan_id`、写入 `scans` 表、将每个目标转换为 `Job{CIDR, Interface, ScanID}` 调用 `mgr.Enqueue`。
   - `StartInterval(ctx, targets, interval)`：若 interval >0，开 ticker 定时调用 `Trigger`。
   - 支持 `Stop()` 用于优雅关闭。
   - 在 `cmd/lanmapper/main.go` 启动流程中：
     - 加载配置后调用 `DetectDefaultCIDRs()`；若配置中提供 `scan_cidr` 则解析追加。
     - 若最终列表为空，log warn 并跳过扫描。
     - 否则 `sched.Trigger(ctx, targets)` 并 `sched.StartInterval`。

3. **API 更新**
   - `server.Start` 注入 scheduler 与默认 targets（供 fallback）。
   - `/api/v1/scans`：
     - 解析 JSON body `{ "cidr": ["192.168.31.0/24"], "interface": "eth0" }`。
     - 校验 CIDR 格式，若为空则使用默认 targets。
     - 调用 `sched.Trigger`，返回 `{ "scan_id": "...", "targets": ["192.168.31.0/24"] }`。
     - manager/scheduler `nil` 时返回 503。

4. **数据存储**
   - 利用现有 `scans` 表（在 migrations 中已经定义）或补充缺失字段；记录 `id, status, started_at, finished_at, targets(JSON)`。
   - `data.Store` 新增 `InsertScan`、`FinishScan`。
   - Runner 完成时可调用 `FinishScan`；若暂时无法感知完成时机，可先在 trigger 后把状态置为 `running`，后续迭代改进。

5. **配置与文档**
   - README/usage：说明“默认自动检测网段；`SCAN_CIDR` 覆盖/追加、`SCAN_INTERVAL` 控制周期”。
   - 提醒需要 `--net=host`/`cap_net_raw`，否则可能检测不到网段。

## 验收标准
- 未设置 `SCAN_CIDR` 时启动容器即可触发至少一次扫描，日志显示所选网段。
- `curl http://localhost:8080/api/v1/scans` 返回新 `scan_id`，并在日志/数据库看到记录。
- 设置 `SCAN_INTERVAL=1m` 时能每分钟触发一次（通过日志确认）。
- `go test ./...` 通过，包括新加的检测/调度测试。
