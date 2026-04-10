# 单设备局域网拓扑扫描 Docker 工具设计

## 1. 背景与目标
- 在一台部署于局域网的 Linux 主机上运行 Docker 容器，通过 `--cap-add=NET_ADMIN` 和（推荐的）`--net=host` 获取底层访问能力。
- 用户只需提供扫描网段或使用默认网段即可，无需额外凭证（可在配置中添加 SNMP community 列表）。
- 工具完成自动发现、拓扑构建、Web UI 展示，以及 PNG/SVG/JSON/CSV/PDF 的报告导出。
- 支持 IPv4 + IPv6，优先覆盖 ARP/ND、ICMP、mDNS/SSDP、SNMP/LLDP/CDP 等协议；保留扩展钩子。

## 2. 成功判定
- 默认配置下运行 `docker run --rm --net=host --cap-add=NET_ADMIN tuobutu/lanmapper:dev` 即可访问 Web UI 并触发扫描。
- 在典型家用/中小企业局域网内（<=1k 设备）10 分钟内生成拓扑，节点覆盖率>90%。
- Web UI 显示实时节点/链路，支持筛选和详情查看；导出按钮可获取 PNG/SVG/JSON/CSV/PDF。
- 所有扫描和报告数据落地在 `data/` 卷的 SQLite 中，可保留多次扫描历史。

## 3. 架构概览
```
+-------------------+        +-------------------+
| React Web UI      |<------>| REST/WebSocket API|
+-------------------+        +-------------------+
                                     |
                           +-------------------+
                           | Core Scanner (Go) |
                           +-------------------+
                                     |
                             +---------------+
                             | SQLite (data) |
                             +---------------+
```
- Go 单体进程负责 API、WebSocket、任务调度、扫描协议插件、报告生成。
- 前端静态文件随镜像打包，由同一 Go 进程托管。
- 数据存储使用 SQLite + 内嵌图缓存（内存 map + Bolt-like 结构），并周期写盘。

## 4. 模块划分
1. **Scanner Manager**：负责任务队列、并发控制、接口检测、速率限制。
2. **Protocol Runners**：
   - `arp_nd`：通过 `gopacket` + raw socket 进行 ARP(IPv4)/ND(IPv6) 探测。
   - `icmp_syn`：ICMP echo + TCP SYN（可选端口列表）验证在线状态。
   - `snmp_l2`：基于 `gosnmp` 读取 `sysDescr`, `ifTable`, `LLDP-MIB`, `CISCO-CDP-MIB`, `BRIDGE-MIB`。
   - `service_discovery`：mDNS、SSDP/UPnP、NetBIOS、HTTP banner，以 UDP 多播/广播方式。
3. **Topology Builder**：
   - 维护 `Device`, `Interface`, `Link` 结构体。
   - 合并多源信息形成置信度评分，解决模糊链路（使用加权 Union-Find + 图推理）。
4. **Data Service**：封装 SQLite 操作，提供 CRUD、分页、索引。表：`devices`, `interfaces`, `links`, `scans`, `reports`, `events`。
5. **API Layer**：使用 `Fiber` 或 `Echo`，REST + WebSocket；提供 `/api/v1` 命名空间。
6. **Report Engine**：Go templates 输出 JSON/CSV；Chromedp/headless Chrome 渲染 HTML → PDF/PNG/SVG。
7. **Frontend**：React + Vite + Tailwind；拓扑图使用 `vis-network` + Dagre 及缩略图；状态面板用 `Recharts`。
8. **Config & Secrets**：支持 `config.yaml` + 环境变量；默认 SNMP community 列表可在 `config/snmp.yaml` 配置。

## 5. 数据模型
- `Device`：`id`, `ip4`, `ip6`, `mac`, `vendor`, `type`, `os`, `protocols`, `last_seen`, `tags`, `confidence`。
- `Interface`：`id`, `device_id`, `if_name`, `mac`, `speed`, `vlan`, `is_uplink`。
- `Link`：`id`, `a_device`, `a_if`, `b_device`, `b_if`, `media`, `speed`, `confidence`, `source`。
- `Scan`：`id`, `status`, `started_at`, `finished_at`, `config_snapshot`, `stats`。
- `Report`：`id`, `scan_id`, `type (json/csv/pdf/png/svg)`, `path`, `created_at`。
- `Event`：用于日志/警告（默认凭证、异常设备等）。

## 6. 扫描流程详细
1. **初始化**：读取 `config.yaml`，列出本机活跃接口、CIDR；创建 `Scan` 记录。
2. **广播/被动监听**：
   - 加入接口多播组，监听 mDNS(5353)、SSDP(1900)、NetBIOS、LLDP。
   - 收到的公告直接转化为 `Device`。
3. **主动扫描**：
   - ARP/ND：对每个网段分批 256 IP 发送，结合 libpcap 捕获响应。IPv6 采用 ICMPv6 Neighbor Solicitation。
   - ICMP/TCP：对所有在线/疑似在线 IP 做存活检测；对常见端口（22/80/443/161）发送 SYN。
4. **信息采集**：
   - SNMP：尝试默认 community + 配置列表，读取 LLDP/CDP/Bridge/MAC 表。
   - HTTP：抓取 Server/Banner；UPnP 获取 friendlyName。
5. **拓扑推断**：
   - 利用 LLDP/CDP 链接直接建边；
   - MAC 表 + 接口 VLAN 交叉比对确定接入设备；
   - 对未解析链路使用 Heuristic：相邻 IP、同交换机接口等；
   - 记录置信度供 UI 展示（颜色/虚线）。
6. **结果落地**：统一写 Data Service（批处理）。
7. **通知前端**：通过 WebSocket 推送 `scan_progress` 和 `topology_update` 消息。
8. **报告生成**：扫描完成触发导出任务，生成 JSON/CSV；PDF/PNG/SVG 使用 chromedp 截取拓扑图+统计页面。

## 7. Web UI 功能
- 登录页：可选密码（默认禁用）；提醒只在可信 LAN 使用。
- 仪表盘：
  - 当前扫描进度、设备计数、异常告警（默认凭证、不可达）
  - 最近扫描历史列表，可回看报告。
- 拓扑视图：力导布局 + 分层布局切换；节点类型颜色、鼠标悬浮详情；筛选（设备类型、厂商、置信度）。
- 设备详情页：展示接口、协议响应、历史事件、导出单设备报告。
- 报告中心：列出可下载文件；提供重新生成按钮。

## 8. 配置与运行
- 环境变量：`SCAN_CIDR`, `SNMP_COMMUNITIES`, `SCAN_INTERVAL`, `HTTP_PORT`, `UI_LANG` 等。
- 也可挂载 `data/config.yaml`：
  ```yaml
  cidr: ["192.168.0.0/16"]
  snmp:
    enabled: true
    communities: ["public", "private"]
    timeout_ms: 1500
  icmp:
    rate_per_sec: 200
  ```
- Dockerfile：多阶段，最终镜像 ~150MB。启动脚本检测 `data/lanmapper.db`，若无则初始化 schema 并运行迁移。

## 9. 安全与性能
- 访问控制：可通过 `LM_ADMIN_TOKEN` 环境变量启用简单密码；未来可接入 OIDC。
- 日志：结构化 JSON，可输出到 stdout；提供 `--log-level`。
- 性能：
  - 扫描任务按接口分区并行，默认 5*CPU 并发；
  - Raw socket + libpcap buffer 调优（BPF 过滤）；
  - SNMP 设定全局速率（如 50 req/s），防止设备过载。
- 隐私：默认禁用互联网访问；远程导出需用户自行复制。

## 10. 测试计划
1. **单元测试**：协议解析（SNMP → 结构体）、拓扑推理算法、配置加载、报表模板。
2. **仿真测试**：使用 ContainerLab/kind 构建多交换机拓扑，模拟 LLDP/SNMP 响应；验证 IPv4/IPv6 覆盖。
3. **端到端**：docker-compose 带虚拟节点（如 Open vSwitch + Alpine 主机）跑一次扫描，断言 API/WS/导出成功。
4. **性能基准**：1000 节点虚拟拓扑内完成扫描<10 分钟，内存占用<1GB。
5. **安全测试**：确认禁用协议或黑名单时不会发送对应探测；默认凭证提示准确。

## 11. 风险与缓解
- **SNMP 凭证不可用**：提供配置让用户自定义列表；若无则依赖被动方式。可在 UI 提示“提供 community 以提高准确度”。
- **拓扑推断误差**：UI 显示置信度及说明来源（LLDP、MAC、Heuristic）；允许用户手工编辑并保存。
- **Chromedp 生成报告缓慢**：支持异步排队，限制并发；若失败可回退为纯 HTML 输出。
- **HOST 网络不可用**：提供 `SCAN_INTERFACE` 设置 + `setcap cap_net_raw+ep`，在非 host 网络下通过原始套接字仍可工作，但广播捕获受限（文档中提示）。

## 12. 里程碑
1. **MVP (2 周)**：核心扫描 + SQLite + REST API + 简单 React 拓扑视图 + JSON/CSV 导出。
2. **报告增强 (1 周)**：PDF/PNG/SVG、Chromedp pipeline、UI 导出控件。
3. **优化与扩展 (2 周)**：SNMP/LLDP 覆盖优化、置信度算法、UI 过滤/历史、配置界面。
4. **打包发布 (0.5 周)**：Docker 镜像、CI 构建、文档、示例 docker-compose。

