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
Problem 通过 canonical formatter 展示 params 和 operationId。真实 Run 诊断中模型在未创建修改候选时调用
`workflow_compile`/`workflow_preview` 曾触发 `ErrNoProposal` 并被折叠为 `ai.authoring.failed`；现已允许这两个工具
分别检查当前持久 Source、返回空变更，并保留候选存在时的原有验收语义。同一生产数据、真实 Run 和 Codex 配置
已完整执行 `run_get → workflow_inspect → workflow_compile → workflow_preview` 并成功返回无需修改的诊断结论；
`run_get` 现在还提供从 Run 开始到当前时刻或终态的 `elapsedMilliseconds`，供 AI 明确报告运行时长。
`task check` 通过。最新生产 EXE 中由 owner 发起同场景的最终 UI 验收仍待完成。

## Next

构建并启动最新生产 EXE，通过[AI 提案侧栏](../../../frontend/src/app/editor/AIWorkflowReviewPanel.vue)在原工作流发送
“帮我在最后加一个按键节点 按F键”；确认侧栏生成可审查候选且失败表面展示结构化 Problem 与 operationId 后，
再关闭本 Work。

## References

- [稳定上下文](context.md)
- [错误契约开发指南](../../knowledge/errors/error-contract.md)
- [上一阶段 AI Run 诊断工作](../ai-run-diagnostics/index.md)

## Progress

- 2026-09-03 根据 owner 建议为 AI Run evidence 增加结构化 `elapsedMilliseconds`：运行中使用当前时刻，终态使用
  `endedAt`，并通过 Run timing accessor 测试覆盖持久时间投影。
- 2026-09-03 复现 owner 的真实 `ai.authoring.failed`：Run 读取与 Source 检查成功后，模型在无候选时调用
  `workflow_compile`，被 `ErrNoProposal` 中止。补齐基线编译/空变更预览语义和回归测试；同一真实 Run smoke
  随后成功完成 6 轮诊断，AI eval 8/8 approved，`task check` 通过。
- 2026-09-03 精确真实 smoke 证明 12 次迭代预算在候选完成前耗尽；修复目录发现提示、预算错误投影、持久参数和
  UI 关联 ID，并更新陈旧 WebView smoke。修复后同场景 18 次迭代成功；`task check`、`task webview:smoke` 通过。
- 2026-09-03 将 AI 提案单轮步骤上限改为持久用户设置，范围 8–64、默认 24；设置页和耗尽错误均解释其为
  Yotta 防循环限制而非账户余额。相关设置、bindings、UI 和 WebView 验证通过。
- 2026-09-03 建立独立验收 Work。记录当前真实 UI 仍失败，禁止继续以错误显示改善、mock、旧请求或隔离样本
  成功宣称 AI 提案完成。
