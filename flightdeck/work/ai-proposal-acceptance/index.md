# AI 提案验收

## Goal

让工作流编辑器中的 AI 提案在真实生产 WebView、真实工作流和真实 Codex 配置下稳定完成普通问答、Run 诊断、
数值修改与“新增节点 + 配置输入 + 连接”结构修改；失败时必须在对应对话中保留可翻译 Problem 和 operationId，
不得再显示无对象、无原因的通用错误。

## Status

Open

## Current

本阶段仍未通过真实 UI 验收。owner 在生产 EXE 中对工作流 `3ddbf6b1-6aff-499a-93d5-89f75429c4b3` 请求
“帮我在最后加一个按键节点 按F”，侧栏仍显示 `transport.unstructured_failure`；持久对话记录的真实 Problem 是
`ai.authoring.tool_input_invalid`，operationId 为 `fa703258-ebf5-4b94-96e3-958500a10347`。不得用先前阈值诊断或
隔离副本成功代替这条真实 UI 验收。

当前代码已具备 workflow-scoped 对话持久化、Codex 隐藏进程、Run evidence、窄 authoring tools、MCP
`workflow_set_input_value` 和真实 smoke CLI。隔离 profile 上同类“回放精准轨迹后新增按 F 节点”已能依次执行
`workflow_add_node → workflow_set_input_json → workflow_connect → workflow_compile → workflow_preview`，但生产
UI 仍需重新验证，错误表面也需要保证优先展示持久 Problem 而非 transport fallback。

## Next

在不要求 owner 再提供信息的前提下，用[真实 smoke 工具](../../../cmd/ai-authoring-smoke/main.go)和
[AI authoring manager](../../../internal/aiauthoring/manager.go)复现当前精确请求，随后通过
[AI 提案侧栏](../../../frontend/src/app/editor/AIWorkflowReviewPanel.vue)完成生产 WebView 的同场景验收；只有真实
UI 生成可审查候选且对话记录一致时才可关闭本 Work。

## References

- [稳定上下文](context.md)
- [错误契约开发指南](../../knowledge/errors/error-contract.md)
- [上一阶段 AI Run 诊断工作](../ai-run-diagnostics/index.md)

## Progress

- 2026-09-03 建立独立验收 Work。记录当前真实 UI 仍失败，禁止继续以错误显示改善、mock、旧请求或隔离样本
  成功宣称 AI 提案完成。

