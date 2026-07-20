# R0 dirty-worktree ownership map

## Rule

主工作树在 R0 前已经包含大量未提交实现。以下归属只说明审查和恢复责任，不表示代码正确、阶段完成或允许覆盖用户改动。开始对应 Slice 时必须先读当前 diff，按 intent 保留；无法确认的行保持不动并记录问题。

## Current groups

| Group | Paths | Intended owner | Current treatment |
| --- | --- | --- | --- |
| RPC/error projection | `contracts/wails-rpc.json`、`frontend/src/lib/backend.ts`、Run timeline/i18n | Slice 30 | 先审查是否仍吞错/自动 toast；不以现有改动为完成证据 |
| automation admission/projection | `internal/admission/*`、`internal/appbootstrap/*` | Slice 31 | 吸收 capability 漏投影修复，但最终删除手工二次事实源 |
| application/provider generation | `internal/application/application.go` | Slice 31 | 审查 generation ownership、lease 与回收，不继续扩 composition root |
| installed automation profile | `internal/automation/installed/*` | Slice 31/34 | 保留 exact/regex、identity 修复；profile seam 归 Slice 31，Win32 native 行为归 Slice 34 |
| Settings/service activation | `internal/services/settings*`、`automation_service*`、`app.go` | Slice 31 | Settings 只保留持久化意图，激活协议迁入 runtime owner |
| Windows target capture | `pkg/winutil/window_*`、`internal/desktopapp/desktop.go` | Slice 31/34 | 保留原始 title/F9/selector 修复并补 native journey |
| Automation Settings UI | `SettingsAutomation.vue/spec.ts`、i18n | Slice 31/34 | 先保留用户改动；schema/editor ownership 在 Slice 31 重构，布局/native UX 在 34 验收 |
| key chord authoring | `KeyChordValueEditor.vue`、`keyChord.*`、`WorkflowInputBindingEditor*` | Slice 34/35 | 作为 Press Keys 创作入口验收，不单独宣称完成 |
| Workflow editor/run UI | `WorkflowEditorView.vue`、`RunTimelinePanel.vue`、runtime panel tests | Slice 30/35 | 只在对应边界修改，避免巨型页面继续吸收生命周期 |

## Untracked ownership

- `architecture-health-audit.md`、本 Topic references/references/slices 属于 Flightdeck 计划与恢复证据。
- `KeyChordValueEditor.vue`、`WorkflowInputBindingEditor.spec.ts`、`keyChord.ts/test.ts` 属于既有 Press Keys authoring 批次，不是 R0 新写代码。
- Slice 26/27/28 文档属于先前恢复规划；后续由 Slice 31 和新执行 Slice registry 接续，不删除其中的真实诊断证据。

## Invalidation

任何上述路径被提交、还原、重写或由其他任务修改后，开始对应 Slice 前重新生成此归属表。工作树干净与否不改变 capability ledger 的证据要求。
