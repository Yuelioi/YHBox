# AI Agent bounded tool runtime

Status: current

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

Provider Outcome 已能表达 tool-call，ModelProfile 已声明 tool capability；Generate/Extract runtime 当前仍把任何非 completed finish 当失败。ToolSet artifact seam 已建立，但尚无 trusted ToolSet registry、continuation loop、approval 或 Agent Node Contract。

## Out of scope

- 长期 conversation Session。
- 任意 ambient filesystem/network/process/window 工具。
- eval gate 与 authoring review UI。
- 第三方 plugin host 实现。

## Result

Current。先冻结 provider-neutral AgentTurn/ToolCall/RunBudget/ToolExecutor contract，再增加内置受限工具与 provider-native continuation mapping。
