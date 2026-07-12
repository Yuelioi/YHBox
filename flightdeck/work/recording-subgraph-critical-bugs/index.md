# Index — 录制与子图关键故障

## State

两个高优先级故障均已定位并修复，自动化回归与扩大验证通过，等待用户本地 smoke：

1. 简易录制无法记录鼠标点击坐标。
2. 折叠为子图后，子图内部内容变为空。

- 录制转换器仍写旧 `XRatio/YRatio`，而 `ClickAt` 已只读结构化 `Point`。
- 折叠流程的空子图可能先进入 store；update 后普通 `mergePool` 会保留同 ID 空壳，遮住后端完整快照。

## Next

- 用户本地 smoke：简易录制一次非中心位置点击，确认生成 `ClickAt.Point` 坐标正确。
- 用户本地 smoke：选中节点折叠为子图并立即双击进入，确认节点与边存在；保存重开后再确认一次。

## Read now

- `flightdeck/knowledge/nodes/recording-schema-cascade.md`
- `flightdeck/knowledge/subgraph/merge-pool-preserves-created-empty-shell.md`

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

Current:

- 等待用户本地 smoke。

Verified:

- `go test ./internal/services/recording -count=1`
- `frontend: pnpm test` — 68 files / 528 tests
- `frontend: pnpm build` — 成功；仅既有大 chunk warning

## Open questions

- 用户本地实际录制与 Wails 子图 changed 事件时序是否与自动化回归一致。
