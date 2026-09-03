# AI 提案验收

## Goal

让工作流编辑器中的 AI 提案在真实生产 WebView、真实工作流和真实 Codex 配置下稳定完成普通问答、Run 诊断、
数值修改与“新增节点 + 配置输入 + 连接”结构修改；失败时必须在对应对话中保留可翻译 Problem 和 operationId，
不得再显示无对象、无原因的通用错误。

## Status

Open

## Current

已用生产数据目录、真实工作流 `3ddbf6b1-6aff-499a-93d5-89f75429c4b3` 和真实 Codex 配置复现根因：模型在目录
发现期间消耗 12 次迭代，完成新增节点和设置 `F` 后因 `ai.ErrAgentBudgetExceeded` 在连接、编译和预览前终止；
service 将其折叠为 `ai.authoring.failed`，侧栏又把持久 Problem 错误降级成“发生未知错误”。

当前修复已引导英文稳定目录词、将迭代余量增至 24、投影专用可重试 `ai.authoring.budget_exhausted`，并让持久
Problem 通过 canonical formatter 展示 params 和 operationId。同一真实 smoke 已在 18 次迭代完整执行
`workflow_add_node → workflow_set_input_json → workflow_connect → workflow_compile → workflow_preview` 并生成候选；
步骤上限现可在“设置 → AI 模型 → 提案执行”中按 8–64 配置，并明确说明它不是订阅或 API 余额。
`task check` 与 `task webview:smoke` 均通过。最新生产 EXE 中由 owner 发起同场景的最终 UI 验收仍待完成。

## Next

构建并启动最新生产 EXE，通过[AI 提案侧栏](../../../frontend/src/app/editor/AIWorkflowReviewPanel.vue)在原工作流发送
“帮我在最后加一个按键节点 按F键”；确认侧栏生成可审查候选且失败表面展示结构化 Problem 与 operationId 后，
再关闭本 Work。

## References

- [稳定上下文](context.md)
- [错误契约开发指南](../../knowledge/errors/error-contract.md)
- [上一阶段 AI Run 诊断工作](../ai-run-diagnostics/index.md)

## Progress

- 2026-09-03 精确真实 smoke 证明 12 次迭代预算在候选完成前耗尽；修复目录发现提示、预算错误投影、持久参数和
  UI 关联 ID，并更新陈旧 WebView smoke。修复后同场景 18 次迭代成功；`task check`、`task webview:smoke` 通过。
- 2026-09-03 将 AI 提案单轮步骤上限改为持久用户设置，范围 8–64、默认 24；设置页和耗尽错误均解释其为
  Yotta 防循环限制而非账户余额。相关设置、bindings、UI 和 WebView 验证通过。
- 2026-09-03 建立独立验收 Work。记录当前真实 UI 仍失败，禁止继续以错误显示改善、mock、旧请求或隔离样本
  成功宣称 AI 提案完成。
