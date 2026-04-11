# 使用指南

## 运行参数

- `HTTP_PORT`: Web 服务端口，默认 8080。
- `DATA_DIR`: 数据存储目录，默认 `/app/data`。
- `SCAN_CIDR`: 逗号分隔的 CIDR 列表（默认留空，使用容器默认路由接口自动探测到的网段）。
- `SCAN_INTERVAL`: 周期扫描间隔，如 `30s`、`5m`、`1h`，默认 `0s` 表示仅启动时扫描一次。
- `SNMP_COMMUNITIES`: SNMP v2c community 列表。
- `ADMIN_TOKEN`: 若配置，则所有 `/api/v1` 请求需加 `X-Admin-Token` 头。

可将上述写入 `data/config.yaml`，示例：

```yaml
http_port: 8080
data_dir: ./data
scan_cidr:
  - 192.168.0.0/16
snmp_communities:
  - public
  - private
scan_interval: 5m
admin_token: changeme
```

## 常用命令

```bash
# 构建前端并嵌入
npm --prefix webapp install
npm --prefix webapp run build

# 后端测试
go test ./...

# 本地运行
HTTP_PORT=8080 go run ./cmd/lanmapper
```

## API 触发扫描 & 导出

```bash
# 列出设备
curl http://localhost:8080/api/v1/devices

# 触发扫描（不带 body 时使用默认 targets）
curl -X POST http://localhost:8080/api/v1/scans

# 指定 CIDR 和接口触发扫描
curl -X POST http://localhost:8080/api/v1/scans \
  -H "Content-Type: application/json" \
  -d '{"cidr":["192.168.1.0/24"],"interface":"eth0"}'

# 触发报告（返回文件路径）
curl -X POST http://localhost:8080/api/v1/reports
```
