# 前端容错实现 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 保障 Dashboard 与 Topology 在 API 返回空/错误时仍能渲染，提供空态提示并重新构建/交付容器。

**Architecture:** 在 React 组件层做防御式编程，使用 `Array.isArray`、加载/错误状态与空态 UI；保持现有 `vis-network` 拓扑逻辑但仅在有数据时初始化；流程末尾重新构建前端、Go 二进制与 Docker 镜像并暴露公网链接。

**Tech Stack:** React 18 + Vite、TypeScript、@tanstack/react-query、Go 1.23、Docker、ssh 隧道（localhost.run）。

---

### Task 1: Dashboard 数据容错与空态

**Files:**
- Modify: `webapp/src/components/Dashboard.tsx`
- Modify: `webapp/src/components/Dashboard.css`（如需样式）

- [ ] **Step 1:** 引入 `safeArray` helper 和加载/错误判断

```tsx
const devices = useMemo(() => (Array.isArray(devicesQuery.data) ? devicesQuery.data : []), [devicesQuery.data]);
const isLoading = devicesQuery.isLoading;
const isError = devicesQuery.isError;
```

- [ ] **Step 2:** 渲染加载/错误提示

```tsx
if (isLoading) {
  return <div className="dashboard state">正在扫描网络…</div>;
}
if (isError) {
  return <div className="dashboard state error">获取设备数据失败</div>;
}
```

- [ ] **Step 3:** 空态 UI

```tsx
if (!devices.length) {
  return (
    <div className="dashboard empty">
      <h3>暂无设备</h3>
      <p>扫描完成后会自动刷新</p>
    </div>
  );
}
```

- [ ] **Step 4:** 根据 `stats` 渲染正常卡片；必要时在 CSS 中添加 `.dashboard.state`, `.dashboard.empty` 等类。

- [ ] **Step 5:** `npm --prefix webapp run lint`（若配置）或 `npm --prefix webapp run typecheck` 确保无 TS 报错。

### Task 2: Topology 容错与防空初始化

**Files:**
- Modify: `webapp/src/components/TopologyView.tsx`
- Modify: `webapp/src/components/TopologyView.css`（若需要）

- [ ] **Step 1:** 确保 `devices`、`links` 使用 `Array.isArray`

```tsx
const devices = Array.isArray(devicesQuery.data) ? devicesQuery.data : [];
const links = Array.isArray(linksQuery.data) ? linksQuery.data : [];
```

- [ ] **Step 2:** 添加加载/错误状态，与 Dashboard 类似。

- [ ] **Step 3:** 当 `devices.length === 0` 时渲染空态，并避免创建 `Network` 实例。

```tsx
if (!devices.length) {
  if (networkRef.current) {
    networkRef.current.destroy();
    networkRef.current = null;
  }
  return <div className="topology empty">暂无拓扑数据</div>;
}
```

- [ ] **Step 4:** `useEffect` 中先返回空数组，只有在 `devices.length > 0` 时构建 `DataSet`、调用 `setData`。

- [ ] **Step 5:** 如需要，为空态/状态提示增加 CSS。

### Task 3: 前端构建与 Go 编译

**Files/Commands:**
- Run: `npm --prefix webapp install`
- Run: `npm --prefix webapp run build`
- Run: `go test ./...`
- Run: `go build ./cmd/lanmapper`

- [ ] **Step 1:** `npm --prefix webapp install`
- [ ] **Step 2:** `npm --prefix webapp run build`（输出 `ui/dist`）
- [ ] **Step 3:** `go test ./...`
- [ ] **Step 4:** `go build ./cmd/lanmapper`

### Task 4: Docker 重建与运行

**Files/Commands:**
- Run: `docker build -t lanmapper .`
- Run: `docker stop lanmapper || true`
- Run: `docker run -d --rm --name lanmapper --net=host --cap-add=NET_ADMIN -e HTTP_PORT=8080 lanmapper`

- [ ] **Step 1:** `docker build -t lanmapper .`
- [ ] **Step 2:** `docker stop lanmapper || true`
- [ ] **Step 3:** `docker run -d --rm --name lanmapper --net=host --cap-add=NET_ADMIN -e HTTP_PORT=8080 lanmapper`
- [ ] **Step 4:** `docker logs -n 50 lanmapper`

### Task 5: 公网隧道与验证

**Commands:**
- Run: `ssh -oStrictHostKeyChecking=no -R 80:localhost:8080 nokey@localhost.run`
- Validate: 浏览器访问给出的 URL，确认 Dashboard/Topology 正常或空态。

- [ ] **Step 1:** 运行 SSH 隧道命令，记录 URL。
- [ ] **Step 2:** 访问 URL，检查无空白页、控制台无报错。
- [ ] **Step 3:** 截屏或描述验证结果，供用户参考。

---

## 自检
- 覆盖 spec 的“前端容错”“空态提示”“重建交付”全部要求。
- 无 TODO/模糊指令。
- 类型命名与组件一致。
