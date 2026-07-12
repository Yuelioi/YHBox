# Index — 录制资产生命周期

## State

实施阶段。后端录制生命周期契约已完成：停止生成内存 pending，Finalize 才落库，Discard/Cancel 不产生资产；未引用录制支持预览与删除前引用复检。

目标用户以普通人为主，专业能力渐进披露；设计关键词为清晰、可靠、高效。

## Next

- 生成最新 Wails bindings，并更新前端 backend wrapper。
- 实现 HUD 取消、停止后保存面板与资产库清理入口。

## Read now

- `design.md`
- `.impeccable.md`

## Read if

- `flightdeck/knowledge/build/code-style.md` — 开始修改源码前
- `flightdeck/knowledge/build/build.md` — 运行测试、构建或 Wails smoke 前
- `flightdeck/knowledge/frontend/ui.md` — 修改 Vue/Nuxt UI 组件前
- `flightdeck/knowledge/nodes/recording-schema-cascade.md` — 修改录制产物 schema 或跨层字段时
- `flightdeck/knowledge/subgraph/import-bypasses-container-store-cache.md` — 新增录制资产落盘/刷新路径时

## Progress

Done:

- 已完成用户、目的、范围与设计气质确认。
- 已产出录制资产生命周期设计 brief。
- 已实现并测试 pending/finalize/discard/cancel/cleanup 后端契约。

Current:

- 前端交互实现。

Verified:

- `go test ./internal/services/recording -count=1`
- `go test . ./internal/services/inputclip ./internal/services/container -count=1`

## Open questions

- 是否将 `.impeccable.md` 的 Design Context 同步到 `.github/copilot-instructions.md`。
