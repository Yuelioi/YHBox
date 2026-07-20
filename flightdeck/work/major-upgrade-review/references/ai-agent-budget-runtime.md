# AI Agent bounded tool runtime

Status: completed (d22b5bd5)

## Outcome

新增 provider-native Agent 节点与可审计 tool loop；每次 tool request 都受 exact ToolSet、schema、capability approval 和多维 RunBudget 约束，预算耗尽或不匹配时稳定 fail closed。

## Completion criterion

- Agent Node Contract 固定 prompt/context input、result output、error/status facts、ToolSet 与 budget identity；不接受 ambient tool registry。
- OpenAI/Anthropic tool request/continuation 显式映射稳定 domain contract，不暴露 provider wire transcript。
- Tool arguments/result 均按 manifest schema 校验；tool result 始终属于 untrusted block。
- token、cost、wall time、iteration、tool call 与 parallelism budget 在宿主执行并形成 durable terminal facts。
- 权限扩大或 open-world effect 进入统一 approval/capability seam；模型不能自行授权。
- cancellation、partial provider response、unknown tool、schema violation、budget exhaustion 与 retry ambiguity 有稳定测试；task check 全绿。

## Blocked by

无。ai-prompt-tool-provenance 已由 b674664c 完成。

## Verification

- provider-neutral AgentStart/Continue、ToolResult、RunBudget、BudgetTracker 与 ToolExecutor 已冻结；全部累加器 fail closed 并防溢出。
- OpenAI Responses 支持 stored previous_response_id 与 store:false 完整 native output replay；每轮固定 trusted instructions、exact tools 与 parallel policy。
- Anthropic Messages 保存并原样回传 assistant content blocks（含 thinking/signature/未知扩展字段），再用一条 user message 返回全部 tool_result。
- continuation state copy-on-write；provider/contract 失败不推进 session state，RetryAmbiguous 保留 pending call。
- Resource scope 强制 Agent start/continue 与 Generate/Structured 隔离；ToolSet output schema 与 pending call identity 在 provider boundary 再验证。
- Agent Node 注册 exact built-in pure text_length，无 filesystem/network/process/window ambient authority。
- terminal action 记录 prompt/toolset digest、provider/model/finish、token/cost 与全部 budget counters，不记录 transcript、prompt、tool result 或 secret。
- 2026-07-17 task check 全绿：global coverage 65.8%，internal/ai 74.0%，frontend 27/103，Wails 100 models。

## Out of scope

- 长期 conversation Session。
- 任意 ambient filesystem/network/process/window 工具。
- eval gate 与 authoring review UI。
- 第三方 plugin host 实现。

## Result

d22b5bd5 完成 bounded native Agent runtime、内置 Agent Node、provider continuation、budget/authority/session boundary、settings pricing 与全链路测试。
