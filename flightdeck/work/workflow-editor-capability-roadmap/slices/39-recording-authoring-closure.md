---
slice: "39"
title: 录制与资产创作闭环
status: completed
---

# Slice 39：录制与资产创作闭环

## Outcome / Question

让资源库和编辑器共享唯一 RecordingSession 生命周期；把简易/精准录制恢复为同一 InputClip 的可理解创作流程，并让模板选择有明确完成动作。

## Completion criterion

- `recording:completed` 只进入一个 store/session owner；keep-alive/跨路由不会出现第二个 pending modal 或二次 finalize。
- 资源库入口只保存；编辑器入口保存成功后可插入画布。
- 简易录制恢复 key/button/scroll，支持 action 增删、前后插入、重排、持续时间和 delay 编辑。
- 精准录制保留 move/path/timing，编辑投影折叠轨迹，回放仍走唯一 InputClip runtime。
- 后端 normalize/validate edited actions，拒绝不完整 press/release、非法时间和超预算 payload。
- Template picker 有 selected state、variant 选择与固定确认按钮。
- 关键行为有组件/服务回归测试，不以 source-string test 替代。

## Verification

- recording service/store/component 契约测试。
- canonicalize/edit projection/round-trip/playback 定向 Go tests。
- AssetsView、WorkflowEditorView、AssetPickerModal 组件行为测试。
- 两种入口、两种 mode 和模板选择真机 smoke 在本阶段末批量验收。

## Out of scope

- 不做已保存 clip 的破坏性原地版本覆盖。
- 不把精准采样点全量发送到 Vue 普通数组。
- 不在每个微改动后运行整仓门禁。

## Result

Completed。

- recording store 成为 completion/finalize 的唯一 owner，并携带 library/editor 调用来源；keep-alive 页面不再重复消费 pending。
- 资源库入口只保存，编辑器入口保存后插入 InputClip 节点；后端 finalize 接收并校验可选 edited actions。
- 简易宏恢复 key/click/scroll，支持分页编辑、增删、重排、delay 与 duration；精准录制保留原始轨迹/时序并提供折叠预览。
- Template picker 具有候选选中态、variant 与固定确认动作。
- recording Go tests、store/component tests、前端 189 tests、WebView smoke 和 Windows native recorder smoke 通过。
