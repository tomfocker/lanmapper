# LAN Mapper

单设备部署的 Docker 化局域网拓扑扫描工具。Go 后端负责扫描、拓扑构建、REST/WS API；React 前端展示拓扑与报表；容器只需 `--net=host --cap-add=NET_ADMIN` 即可运行。

## 快速开始

```bash
git clone <repo>
cd lanmapper
npm --prefix webapp install
npm --prefix webapp run build
GOOS=linux GOARCH=amd64 go build ./cmd/lanmapper
```

### Docker 运行

```bash
docker build -t lanmapper .
docker run --rm \
  --net=host \
  --cap-add=NET_ADMIN \
  -v $(pwd)/data:/app/data \
  -e HTTP_PORT=8080 \
  -e SCAN_INTERVAL=5m \
  lanmapper
```

容器启动后会自动检测默认路由接口的网段（例如 `192.168.1.0/24`），立即触发一次扫描，并按 `SCAN_INTERVAL`（默认 `0s`，表示禁用）定期执行。只有在自动检测不到或需要追加额外子网时才需要设置 `SCAN_CIDR`。

访问 `http://localhost:8080` 查看 UI。

## 主要目录

- `cmd/lanmapper`：主程序入口，加载配置、启动扫描和 HTTP 服务。
- `internal/config`：Viper 配置加载。
- `internal/data`：SQLite 存储及迁移。
- `internal/scanner`：ARP/ICMP/SNMP/mDNS 等协议 Runner。
- `internal/topology`：拓扑构建。
- `internal/report`：JSON/CSV 导出。
- `internal/api`：Fiber REST/WebSocket + 静态文件托管。
- `webapp`：React 前端（Vite）。
- `ui/dist`：前端编译后供 Go embed 的静态文件。

## 配置

环境变量或 `data/config.yaml`：

```yaml
http_port: 8080
scan_cidr: ["192.168.0.0/16"]
snmp_communities: ["public", "private"]
scan_interval: 0s
admin_token: "" # 若非空，需在请求头附带 X-Admin-Token
```

- `SCAN_CIDR` 留空时会自动读取容器默认路由接口生成 CIDR 列表；设置后会在自动检测基础上追加/覆盖。
- `SCAN_INTERVAL` 接受 `30s`、`5m`、`1h` 这类 `time.Duration` 字符串，默认为 `0s`（仅启动时扫描一次）。

## API

- `GET /api/v1/devices`：设备列表
- `GET /api/v1/links`：链路
- `GET /api/v1/topology`：拓扑快照
- `POST /api/v1/scans`：立即触发扫描，可选 JSON body：

```json
{
  "cidr": ["192.168.1.0/24"],
  "interface": "eth0"
}
```

  不传 body 时使用自动检测到的默认 targets，返回 `{"scan_id":"...","targets":["192.168.1.0/24"]}`。
- `POST /api/v1/reports`：生成 JSON 报告

## 开发

```bash
# 后端测试
go test ./...

# 前端开发
cd webapp
npm install
npm run dev
```

在根目录运行 `npm --prefix webapp run build` 会将静态文件输出到 `ui/dist`，Go 在编译时通过 `embed` 打包。
