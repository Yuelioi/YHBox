# J1 — 录制与模板上下文连续性

## Goal

让精准录制缺少校准时给出可执行恢复路径，并让节点参数的模板选择旅程能在当前上下文直接捕获、
保存和回填模板。

## Current

完成。校准错误由共享反馈模块提供进入输入校准页的动作；模板选择器的 capture 事件沿
`WorkflowInputBindingEditor → AuthoringSurfaceItem → Inspector → WorkflowEditor` 回到当前节点，随后复用
ScreenPicker 与 `useWorkspaceResource` 保存并绑定。

## Next

无。Stage J 继续 J2。

## Verification

- `pnpm -C frontend exec vitest run src/composables/useRecordingStartFeedback.test.ts src/components/assets/AssetPickerModal.spec.ts src/app/editor/WorkflowInputBindingEditor.spec.ts src/views/WorkflowAuthoringFoundations.spec.ts`
- `pnpm -C frontend typecheck`
- `pnpm -C frontend i18n:check`

## Acceptance

- 精准录制收到 `RECORDING_CALIBRATION_REQUIRED` 时，用户能在错误面直接进入当前目标的校准旅程，
  或明确切换到可用录制方式；失败不丢失编辑器上下文。
- 从节点参数打开模板选择器时，可直接启动模板捕获；保存后返回原参数并选中该模板。
- 既有资源 Dock、选择已有模板、取消和失败路径不回归。
- 定向前端测试通过；无并行资源状态或第二套录制状态。

## Evidence

- [真机反馈 FD-01/FD-02](../references/real-device-feedback-2026-07-20.md)
- [UI / 交互调研](../references/real-device-feedback-research-2026-07-20.md)
