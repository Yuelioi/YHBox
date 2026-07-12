# Index — 录制资产生命周期

## State

设计阶段。用户确认三个需求：录制中可取消；可批量清理所有未引用的录制资产；停止录制后填写名称、分类和标签，再决定保存或丢弃。

目标用户以普通人为主，专业能力渐进披露；设计关键词为清晰、可靠、高效。

## Next

- 用户确认 `design.md`。
- 确认后进入实现：先固化 pending/finalize/cancel/cleanup 后端契约与测试，再接 HUD、保存面板和资产库清理入口。

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

Current:

- 等待用户确认设计 brief。

Verified:

- 尚未进入实现验证。

## Open questions

- 是否将 `.impeccable.md` 的 Design Context 同步到 `.github/copilot-instructions.md`。
