# AI 提案验收上下文

## Product contract

- AI 提案是每个 Workflow 独立的持久对话侧栏；普通问答、Run 诊断和修改请求共享历史，但不得跨 Workflow。
- 模型通过窄工具表达意图，host 负责生成 typed Authoring Command、候选累积、编译和权限预览；用户明确接受前
  不发布修改。
- 阈值等数值输入使用 `workflow_set_numeric_input`；结构修改优先组合 `workflow_add_node`、
  `workflow_set_input_json` 和 `workflow_connect`。不能向模型注入完整 Authoring Patch schema。
- 本机 MCP 提供 `workflow_set_input_value`，与内部 authoring seam 使用相同的 graph/node/input 语义。

## Acceptance boundary

- 验收对象必须是最新生产 EXE 中的真实 WebView，不是仅 Go manager、provider、Vite 或隔离数据层测试。
- 至少覆盖：中文消息落盘、普通回复、真实 Run 诊断、数值候选、添加按键节点并绑定 `F` 后连接执行流、接受候选
  后编辑器重新加载。
- 每个失败对话必须保存 user message、稳定 `problemId`、`operationId`；UI 展示持久 Problem，不用
  `transport.unstructured_failure` 覆盖已有后端证据。
- “发生未知错误”“操作未完成”不是可接受终态。恢复文案只能指向产品中真实存在的操作或入口。

## Known evidence

- `hello` 曾在真实 UI 成功，证明 Codex 基础对话可用。
- 真实 Run 诊断曾通过 smoke 成功生成候选修订，但 owner 后续真实 UI 请求仍失败，因此不能作为最终验收。
- 中文标题曾因 UTF-8 byte/rune 混用失败；新节点候选曾因 generated UUID/handle 不稳定失败。相关回归已加入，
  但生产 UI 仍是最终判据。
