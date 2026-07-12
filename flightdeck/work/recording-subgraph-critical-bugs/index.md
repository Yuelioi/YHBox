# Index — 录制与子图关键故障

## State

折叠子图已由用户确认修复。录制现场复核确认底层坐标非零，但用户运行的是修复前进程；新生产二进制已构建，等待重启后复录。另修复了 Container `package.json` 空链接对象的 schema 诊断。

1. 简易录制无法记录鼠标点击坐标。
2. 折叠为子图后，子图内部内容变为空。

- 录制转换器仍写旧 `XRatio/YRatio`，而 `ClickAt` 已只读结构化 `Point`。
- 折叠流程的空子图可能先进入 store；update 后普通 `mergePool` 会保留同 ID 空壳，遮住后端完整快照。
- 现场 `sg-1f4a...json` 的旧字段实际为 `(0.1039, 0.4069)` 与 `(0.94375, 0.05556)`，证明 hook/客户区换算正常；该子图由 17:17 启动的旧进程生成。
- `bin/Yotta.exe` 已于 22:32 经 `task build` 重建，当前旧进程必须退出并重新启动才能加载新转换器。
- manifest 的可选 `PackageLink` 原为值类型，`omitempty` 仍输出 `{}`；已改为指针并加序列化回归测试。

## Next

- 退出当前 17:17 启动的 Yotta 进程，从新 `bin/Yotta.exe` 重新启动。
- 打开容器 `6ebbf7d4-8bc9-4ca7-b4e5-7db9119286fb` 并保存一次，让 Store 安全重写 package/graph/installation/lock，确认 `package.json` 不再含四个空链接对象。
- 简易录制一次非中心位置点击，确认新子图写 `Point: {x, y}` 且数值非零。

## Read now

- `flightdeck/knowledge/nodes/recording-schema-cascade.md`
- `flightdeck/knowledge/subgraph/merge-pool-preserves-created-empty-shell.md`
- `flightdeck/knowledge/architecture/go-json-omitempty-struct.md`

## Read if

- `flightdeck/knowledge/build/code-style.md` — 开始修改源码前
- `flightdeck/knowledge/build/build.md` — 运行测试或构建验证前
- `flightdeck/knowledge/subgraph/keepalive-singleton-subgraph-store-stale.md` — 若复现与切换容器或 keep-alive 有关
- `flightdeck/knowledge/subgraph/import-bypasses-container-store-cache.md` — 若磁盘内容存在但内存/界面为空
- `flightdeck/knowledge/subgraph/draft-subgraphs-phantom-field.md` — 若其他子图功能从 draft 读取完整子图

## Progress

Done:

- 已登记两个用户可见症状。
- 已加载录制 schema 级联与子图完整数据源相关既有知识。
- 已建立两条秒级确定性回归测试，并先确认修复前稳定变红。
- 已将录制产物改为 `Point`，并让折叠流程在 update 后回读、覆盖 store 空壳。
- 已从用户现场文件确认坐标采集正常，问题是旧二进制仍写旧 schema。
- 已修复 package manifest 空链接序列化，并完成正式生产构建。

Current:

- 等待用户重启新二进制、保存容器并复录。

Verified:

- `go test ./internal/services/recording -count=1`
- `frontend: pnpm test` — 68 files / 528 tests
- `frontend: pnpm build` — 成功；仅既有大 chunk warning
- `go test ./internal/services/container -count=1`
- `task build` — Wails 107 methods / 0 warnings，生产二进制成功生成

## Open questions

- 新进程复录产物是否写入非零 `Point`。
- Go 全量 `go test ./... -count=1` 本轮在 124 秒工具上限内未结束且无失败输出，未计为通过。
