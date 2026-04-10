# LAN Mapper 前端容错与拓扑空态改进设计

## 背景
- 当前部署方式：Go 后端 + React/Vite 前端通过 `ui/dist` 静态资源嵌入二进制，再由 Docker 容器运行。
- 运行中的容器在首次访问 `/` 时出现空白页，浏览器控制台报错 `Cannot read properties of null (reading 'length')`，定位到前端在 API 尚未返回数据时直接对 `null` 调用数组方法。
- 用户期望：单设备部署即可扫描局域网、前端 UI 能正确展示或提示即便数据为空，同时继续提供 Docker 镜像和公网测试链接。

## 目标
1. 使 Dashboard、Topology 等组件在 API 返回 `null`/`undefined`/非数组时仍能渲染，不出现运行时错误。
2. 在未发现设备或链路时提供明显的空态提示，而不是空白区域。
3. 保持现有轮询刷新逻辑（30s）与 API 契约，确保后续拓展无需大规模改动。
4. 完成前端重新构建、Go 二进制重新编译、Docker 镜像重建、容器运行与公网隧道发布，便于用户复测。

## 非目标
- 不改动后端 API 返回结构以及数据库层。
- 不引入复杂的全局状态管理库或额外协议支持。
- 不新增登录/权限功能。

## 方案概述
1. **安全数组封装**：在前端组件中通过辅助函数（或直接 `Array.isArray` 判断）把 `devicesQuery.data`、`linksQuery.data` 转换为数组；对 `useQueries` 结果添加类型守卫。
2. **加载与错误状态**：组件读取 `isLoading`、`isError`，提供：
   - 加载中：展示骨架或简短文字，例如“正在扫描网络…”
   - 错误：展示错误消息及重试按钮，调用 `refetch`
3. **空态展示**：
   - Dashboard：在指标卡内显示“暂无数据”，防止显示 0 造成误解。
   - Topology：当 `devices.length === 0` 时展示描述卡片并避免初始化 `vis-network`；否则正常渲染拓扑。
4. **UI 细节**：复用现有 CSS（`dashboard`, `card` 等）或最小新增，避免影响布局；空态提示采用柔和边框、图标可选。
5. **构建与交付**：
   - `npm --prefix webapp install && npm --prefix webapp run build`
   - `go test ./...`、`go build ./cmd/lanmapper`
   - `docker build -t lanmapper .`
   - `docker run --rm -d --name lanmapper -p 8080:8080 lanmapper`
   - `ssh -oStrictHostKeyChecking=no -R 80:localhost:8080 nokey@localhost.run` 获取公网链接

## 风险与缓解
| 风险 | 影响 | 缓解 |
| --- | --- | --- |
| 仍有组件直接访问未定义数据 | UI 重新报错 | 全局搜索 `devicesQuery.data` 等模式，逐一检查 |
| `vis-network` 状态不同步 | 图不刷新 | 在 `useEffect` 检查 `devices.length` 后再更新数据 |
| Docker 镜像缓存旧静态文件 | 用户看到旧 UI | 构建时增加 `npm ci`/`npm install` 并确认 `ui/dist` 时间戳 |

## 验收标准
1. 打开首页即使数据库为空也能看到 Dashboard 空态与拓扑空态文案。
2. 浏览器控制台无 `TypeError`。
3. `docker logs lanmapper` 中无前端资源 404 或 panic。
4. 公网隧道链接可访问页面并展示空态或真实拓扑。
