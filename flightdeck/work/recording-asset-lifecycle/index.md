# Index — 录制资产生命周期

## State

实施完成，等待桌面端视觉/交互 smoke。停止生成内存 pending，Finalize 才落库，Discard/Cancel 不产生资产；未引用录制支持预览与删除前引用复检。

目标用户以普通人为主，专业能力渐进披露；设计关键词为清晰、可靠、高效。

## Next

- 用最新 `bin/Yotta.exe` 冒烟确认 HUD 取消、停止后命名入库、录制资产清理三条路径。
- smoke 通过后归档本 topic。

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
- 已实现 HUD 二次确认取消、停止后必填命名及可选分类/标签/描述。
- 已实现资产库清理预览、默认全选未引用项、删除前引用复检与跳过提示。
- 已生成最新 Wails bindings，并完成 production build。

Current:

- 等待 Windows 桌面端视觉与交互 smoke（当前会话无可用 in-app browser 实例）。

Verified:

- `go test ./internal/services/recording -count=1`
- `go test . ./internal/services/inputclip ./internal/services/container -count=1`
- `go test ./... -count=1`
- `cd frontend && pnpm test`（68 files / 529 tests）
- `cd frontend && pnpm typecheck`
- `cd frontend && pnpm i18n:check`（2717 keys）
- `cd frontend && pnpm build`
- `task build`（112 methods / 0 warnings，产物 `bin/Yotta.exe`）

## Open questions

- 无；仅剩本机 smoke。
