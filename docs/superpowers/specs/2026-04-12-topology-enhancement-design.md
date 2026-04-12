# 拓扑增强设计（LLDP/SNMP + 类型识别）

## 背景
- MVP 已可将 ARP/ICMP/mDNS 发现写入 SQLite，但设备缺乏类型/名称、链路数据为空。
- 用户希望 UI 能看到至少基础设备分类（路由器/交换机/终端）以及部分链路关系，哪怕是推断出来的。
- 当前 Runner 结构已支持注入 Recorder，可继续扩展协议实现。

## 目标
1. 新增 SNMP 轮询能力：读取 LLDP/Bridge/IF-MIB，提取接口、邻居、sysName/sysDescr。
2. 基于 MAC OUI、mDNS/NetBIOS、SNMP sysObjectID 推断设备类型/名称并持久化。
3. 生成链路记录（links 表），至少覆盖 LLDP 邻居、Bridge FDB 推断、默认网关推断。
4. API/UI 仍复用现有接口（devices/links/topology），前端无需改动即可看到新的字段。

## 范围
- **包含**：新增 SNMP Runner 能力、OUI 映射、链路推断逻辑、数据模型/Recorder 扩展、测试与文档。
- **不包含**：前端 UI 改版、历史趋势、云端可视化、认证鉴权增强。

## 架构概览
```
Scheduler → Manager → (ARP/ICMP/mDNS/SNMP runners)
                                 ↓
                       DiscoveryRecorder
                ↙ device enrich   ↘ link infer
         data.Store.UpsertDevice   data.Store.UpsertLink
                      ↓                      ↓
                 API / UI / Reports consume enhanced data
```

## 关键模块

### 1. SNMP Runner 扩展
- 使用 `gosnmp` 拉取：
  - `1.0.8802.1.1.2` (LLDP) → `lldpRemSysName`, `lldpRemSysDesc`, `lldpRemPortDesc`.
  - `1.3.6.1.2.1.17.2.4` (Bridge FDB) → MAC→Port 映射，结合 `ifDescr`.
  - `1.3.6.1.2.1.1.2` / `1.3.6.1.2.1.1.5` (sysObjectID/sysName) → 设备类型/名称。
- 轮询逻辑：
  1. 对每个目标 IP/社区尝试 SNMP，成功后缓存 session。
  2. 解析 LLDP 邻居并通过 Recorder 记录 LinkObservation（本地设备 <-> neighbor MAC/IP）。
  3. 将 sysName/sysDescr 写入 DeviceObservation（字段：Hostname/Vendor/Type）。
  4. 对 Bridge FDB 产出的 MAC，结合已有 DeviceObservation（MAC/接口）推导“端口挂载了哪些终端”，生成 link。
- 错误处理：每个 OID 拉取失败仅记录 warn，不阻断其他 OID。

### 2. OUI & 被动推断
- 引入轻量 OUI 数据表（压缩 CSV -> Go embed map），通过 MAC 前 24bit 映射 vendor→类型提示（"Apple", "Cisco" 等）。
- `mdns` Runner 若发现 `_workstation`/`_printer` 等服务，也可附带 `DeviceObservation.TypeHint`。
- Recorder 在写入 `models.Device` 时若已存在类型则保留，否则根据 Source/OUI/服务填充 `Type`.
- 对没有 SNMP 的网络：根据 ARP 表和默认网关 IP 推断一个“核心节点”，并为每个终端写入 link（终端→网关）。

### 3. Recorder & 数据层修改
- 扩展 `DeviceObservation` 结构：新增 `SysObjectID`, `Services []string`, `TypeHint`.
- Recorder 在 `handleDevice` 中合并新的字段，写入 `models.Device` 中的 `Type`、`Vendor`、`Hostname`。
- `LinkObservation` 增加 `Kind`（lldp / arp_gateway / bridge），便于前端区分来源 & 置信度。
- 如有需要，为 links 表新增 `kind` 列（可通过 Migration 添加，默认 `manual`）。

### 4. 拓扑构建
- `internal/topology.Builder` 目前读取 store 数据，需确保：
  - `ListLinks` 包含新生成的 link。
  - `devices` 中 `Type` / `Vendor` / `Hostname` 字段用于 UI（已有字段 but seldom filled）。
- 可考虑添加“默认网关”标签：检测 `cfg.ScanCIDR` 或 `routes` 中的 gateway IP，将其 `Type` 设为 `router`。

### 5. API/Docs
- API 返回结构已包含 `Device` / `Link` 所需字段，无需新 endpoint。
- README/usage 更新：说明 SNMP 可选，默认 communities，如何添加 OUI 文件。

## 数据流
1. Scheduler 触发 → SNMP Runner 按 CIDR 轮询。
2. Runner 产生 observation：
   - `DeviceObservation{ID=MAC, Hostname=sysName, Vendor=OUI, TypeHint="switch", SysObjectID=...}`
   - `LinkObservation{ADevice=localMAC, BDevice=neighborMAC, Kind="lldp", Confidence=0.9}`
3. Recorder 统一去重写库。
4. Builder & API 读取 enriched 数据 → 前端显示“设备类型”“链路”。

## 测试策略
1. **SNMP Runner 单测**：使用 gosnmp mock（或接口包装）模拟返回值，断言产生的 observation 正确。
2. **Recorder 扩展测试**：验证 Type/Vendor merge、Link kind 记录。
3. **Integration Smoke**：针对 sqlite-temp DB 运行 Runner + Recorder，确认 `/api/v1/topology` 返回含链路的 JSON。

## 风险
- SNMP 依赖设备开启 community；需允许配置超时时间/并发，避免阻塞。
- OUI 数据体积若过大需压缩/懒加载。
- 链路推断错误可能导致 UI 拓扑混乱；需在 LinkObservation 中带 `Confidence`，并在 Builder 中优先显示高置信度。

## 交付物
- 代码：SNMP 扩展、OUI 映射、Recorder/Store 更新、拓扑构建更新。
- 文档：README/usage 补充 SNMP/OUI 使用方式。
- 验证：`go test ./...` + Docker 构建 + 文档说明如何在局域网演示（curl devices/links）。
