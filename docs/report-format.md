# 报告格式

## JSON

`POST /api/v1/reports` 返回生成文件路径，结构如下：

```json
{
  "generated": "2026-04-10T12:00:00Z",
  "devices": [
    {
      "id": "dev1",
      "ipv4": "192.168.1.1",
      "ipv6": "",
      "mac": "aa:bb:cc:dd:ee:ff",
      "vendor": "Cisco",
      "type": "switch",
      "last_seen": "2026-04-10T11:59:30Z",
      "confidence": 0.95
    }
  ],
  "links": [
    {
      "id": "link1",
      "a_device": "dev1",
      "b_device": "dev2",
      "media": "ethernet",
      "speed_mbps": 1000,
      "source": "lldp",
      "confidence": 0.95
    }
  ]
}
```

## CSV

CSV 输出字段：

| 列名 | 说明 |
| --- | --- |
| ID | 设备唯一 ID |
| IPv4 | IPv4 地址 |
| IPv6 | IPv6 地址 |
| MAC | MAC 地址 |
| Vendor | 厂商 |
| Type | 类型 |
| LastSeen | 最近发现时间（RFC3339） |
