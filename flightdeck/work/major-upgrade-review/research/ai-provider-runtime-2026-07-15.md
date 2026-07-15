# Yotta 3.1 provider-native AI runtime：OpenAI Responses 与 Anthropic Messages

> 调研截点：2026-07-15。仅使用 OpenAI、Anthropic/Claude 官方 API 文档、API Reference 与隐私中心。以下“Yotta 应当”是基于这些一手资料得出的架构建议，不是供应商承诺。

## 结论

Yotta 不应继续把两家压成 `Chat(req) -> Text`，也不应再用 prompt JSON、OpenAI-compatible 自动猜测或“强制调用一个虚构工具”模拟结构化输出。3.1 应共享**语义稳定的执行契约**，由两个原生 adapter 分别实现：

- OpenAI：`POST /v1/responses`，消费 typed `output` / streaming events、原生 `text.format.json_schema`、`function_call` / `function_call_output`。
- Anthropic：`POST /v1/messages`，消费 typed content blocks、原生 `output_config.format`、`tool_use` / `tool_result`。
- 共享层只统一“意图、结果、工具调用、终止原因、用量、错误与取消强度”；provider transcript、reasoning/thinking、状态句柄、存储开关和原始错误码必须保留为 provider-specific 数据，不能伪装成相同语义。

OpenAI Responses 返回带 ID 的 typed response，状态可能是 `completed/failed/in_progress/cancelled/queued/incomplete`；不完整原因包括输出上限与内容过滤，生成失败另有 `response.error`。[OpenAI Responses create reference](https://developers.openai.com/api/reference/resources/responses/methods/create) Anthropic Messages 是无服务端会话的消息生成接口，以 content blocks 和 `stop_reason` 表达成功结束、工具调用、截断、暂停或拒绝。[Anthropic Messages reference](https://platform.claude.com/docs/en/api/messages/create) 因此“HTTP 2xx”绝不能等同于“业务完成”。

## 建议冻结的共享契约

下面是语义形状，不要求包名或字段名逐字相同。所有可省略数值都必须是指针/option，避免把 `0` 错当“未设置”；尤其 temperature=0 是有效值，max tokens 未设置也不能由节点层偷偷补成任意默认值。

```go
type GenerateRequest struct {
    AttemptID   string                 // Yotta 生成；一次网络尝试一个 ID
    Model       string                 // provider-scoped opaque ID
    Instructions []ContentPart
    Turns       []Turn
    Tools       []ToolSpec
    ToolChoice  ToolChoice             // auto | none | required | named
    Parallel    ParallelPolicy         // allow | single
    Output      *StructuredOutputSpec  // nil = 普通内容
    Limits      GenerationLimits       // *temperature, *maxOutputTokens, ...
    Retention   RetentionRequirement
    Continuation *OpaqueContinuation   // 只能回到同 provider/model
}

type Outcome struct {
    Provider, RequestedModel, ResolvedModel string
    ProviderRequestID  string               // HTTP 请求追踪 ID
    ProviderResponseID string               // msg_/resp_ 等资源 ID（若有）
    Items        []OutputItem               // tagged union
    Finish       Finish
    Usage        TokenUsage
    Continuation *OpaqueContinuation
}

type OutputItem = TextItem | StructuredItem | ToolCall | RefusalItem
type ToolCall struct { CallID, Name string; Arguments json.RawMessage }
type ToolResult struct { CallID string; Status ToolResultStatus; Content []ContentPart }

type FinishKind string
// completed | tool_calls | max_output | context_limit | stop_sequence |
// refusal | content_filter | paused | cancelled | failed | unknown
type Finish struct { Kind FinishKind; RawProviderReason string }

type TokenUsage struct {
    InputTotal, InputUncached, CacheRead, CacheWrite *int64
    OutputTotal, ReasoningOutput                     *int64
    ProviderExtras                                   json.RawMessage
}

type ProviderFailure struct {
    Stage       FailureStage // transport | http | stream | generation | contract
    Class       FailureClass // invalid_request | authentication | permission |
                             // not_found | conflict | rate_limit | overloaded |
                             // timeout | server | cancelled | unknown
    HTTPStatus  *int
    ProviderCode, ProviderRequestID, Message string
    RetryAfter  *time.Duration
    Retry       RetryDisposition // never | after_hint | new_attempt | ambiguous
    Raw         json.RawMessage
}

type RetentionRequirement string
// provider_default | no_application_state | zero_retention_required
```

关键约束：

1. `StructuredItem` 只能在 provider 报告正常完成且 Yotta 本地再次通过 schema 验证后产生。拒绝、截断和内容过滤都不是结构化成功。OpenAI 把拒绝作为 typed refusal content，把截断/过滤作为 `incomplete`；Anthropic 的拒绝和 `max_tokens` 是 HTTP 200 下的 stop reason，且官方明确说此时结果可能不符合 schema。[OpenAI structured outputs](https://developers.openai.com/api/docs/guides/structured-outputs) [Anthropic structured outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs)
2. `CallID` 是不透明值，只在对应 provider response/turn 内关联结果。OpenAI 的 `function_call.arguments` 是 JSON 字符串，结果用同一 `call_id` 的 `function_call_output`；Anthropic 的 `tool_use.input` 已是对象，结果用 `tool_use_id` 关联。[OpenAI function calling](https://developers.openai.com/api/docs/guides/function-calling) [Anthropic tool handling](https://platform.claude.com/docs/en/agents-and-tools/tool-use/handle-tool-calls)
3. tool execution error 不是 provider failure。Anthropic 原生映射为 `tool_result.is_error=true`；OpenAI 没有等价布尔语义，adapter 应把 Yotta 固定的 `{status:"error", error:{code,message}}` envelope 放进 function output，不能把它升级成网络/模型失败。
4. `OpaqueContinuation` 必须原样保存 provider 状态，且禁止跨 provider/model 使用。OpenAI 手工续轮需要保留 reasoning/function items，或使用 `previous_response_id`；Anthropic 要把 assistant content blocks（包括 thinking/redacted thinking 与 tool use）按原样放回历史，官方会拒绝被修改的 thinking blocks。[OpenAI Responses migration](https://developers.openai.com/api/docs/guides/migrate-to-responses) [Anthropic errors / thinking-block validation](https://platform.claude.com/docs/en/api/errors)
5. 未识别的 provider event/type 不能当成普通完成。保留 raw type/JSON 并以 `unknown` 或 contract failure 结束；OpenAI 明确把新增响应字段和 streaming event type 视为兼容变更。[OpenAI API overview](https://developers.openai.com/api/reference/overview)

### 严格 schema profile

Yotta 应编译一个显式的、版本化的 `yotta.ai-schema/3.1` 子集，而不是让 SDK 静默改写：object 必须 `additionalProperties:false`，所有属性必须列入 `required`，可选值用 nullable union 表示；支持基础 scalar/object/array/enum/`anyOf`，其余 keyword 逐项 capability-check，不能降级成 prompt 约束。OpenAI strict mode 要求 object 禁止额外属性且全部字段 required，并只支持 JSON Schema 子集。[OpenAI supported schemas](https://developers.openai.com/api/docs/guides/structured-outputs) Anthropic SDK 会移除某些不支持约束、把它们改写进 description 后再在本地验证；Yotta 的 Go adapter 不应把这种“模型未受约束、SDK 事后检查”冒充原生严格保证。[Anthropic schema transformation](https://platform.claude.com/docs/en/build-with-claude/structured-outputs)

模型能力必须在编译/预检期确认；不支持 native structured output 或 strict tool use 就明确失败，不保留 prompt fallback。Anthropic 当前原生最终 JSON 是 `output_config.format={type:"json_schema"...}`，strict tool 是工具上的 `strict:true`，两者可以同用；旧 `output_format` 只是迁移期接口。[Anthropic structured outputs](https://platform.claude.com/docs/en/build-with-claude/structured-outputs) OpenAI Responses 的最终 JSON 位于 `text.format`，工具 strict 与最终 JSON 是两个不同开关。[OpenAI Responses create reference](https://developers.openai.com/api/reference/resources/responses/methods/create)

## Provider 映射（不得抹平的差异）

| 语义 | OpenAI Responses | Anthropic Messages | Yotta 规则 |
|---|---|---|---|
| 最终 JSON | `text.format.type=json_schema`, `strict:true` | `output_config.format.type=json_schema` | 都本地复验；严禁 prompt/虚构工具兜底 |
| 工具定义 | `type:function`, `strict:true` | tool `input_schema`, `strict:true` | 使用同一受限 schema profile |
| 工具调用 | output item `function_call`; args 为 JSON string；可一次多个 | assistant `tool_use` block；input 为 object；可一次多个 | 保留列表、顺序和 call ID；不要只取第一个 |
| 工具结果 | `function_call_output` + `call_id` | 紧邻的 user `tool_result` + `tool_use_id`，错误可设 `is_error` | shared result 编译成各自 wire shape；Anthropic 的顺序约束由 adapter 保证 |
| 正常/非正常结束 | response status + typed refusal + incomplete details | `stop_reason`：`end_turn/max_tokens/stop_sequence/tool_use/pause_turn/refusal/...` | 映射到 `FinishKind` 并保留 raw reason；拒绝不是 transport error |
| 并行工具 | `parallel_tool_calls=false` 保证 0 或 1 个 function call | provider 有自己的 parallel tool semantics | shared policy 只表达 allow/single；adapter 做能力验证 |
| 请求 ID | response header `x-request-id`；可传 `X-Client-Request-Id` 供追踪 | 每个响应有 `request-id`；错误 body 也有 `request_id` | 两者都入 attempt journal；client ID 是 correlation，不是幂等键 |
| 服务端状态 | 可 `store`、retrieve、`previous_response_id`、background | Messages 本身是 stateless；客户端重传历史 | 默认使用 Yotta-owned transcript；provider state 放 opaque continuation |

OpenAI streaming 具有 `response.completed`、`response.failed`、`response.incomplete` 等终态事件，usage 通常随最终 response 才完整；runtime 应以终态 reducer 收口，而不是把 SSE EOF 当成功。[OpenAI Responses streaming events](https://developers.openai.com/api/reference/resources/responses/streaming-events) Anthropic SSE 可能在 HTTP 200 之后再发 error event，因此同样不能把握手成功或连接 EOF 当完成。[Anthropic API errors](https://platform.claude.com/docs/en/api/errors)

## Usage 的精确归一化

不能共享一个含义模糊的 `PromptTokens`：

- OpenAI `usage.input_tokens` 是 input 总数，`cached_tokens` 是其中 cache 命中的子集；`output_tokens` 有 `reasoning_tokens` 分解，另有 `total_tokens`。[OpenAI Responses usage](https://developers.openai.com/api/reference/resources/responses/methods/create)
- Anthropic 明确规定总 input 为 `input_tokens + cache_creation_input_tokens + cache_read_input_tokens`；`output_tokens` 是计费权威总数，thinking 是其只读分解。[Anthropic Go Messages reference](https://platform.claude.com/docs/en/api/go/messages)

因此 Yotta 的 `InputTotal` 对 OpenAI 直接取 `input_tokens`，对 Anthropic取上述三项之和；同时分别保留 `CacheRead` 与 `CacheWrite`。所有值允许 absent，流中只发布单调 snapshot，最终 journal 再冻结。成本不能只靠 token 总数推导，价格、cache read/write、server tool 与 service tier 都需 provider/model 快照，暂留 `ProviderExtras`。

## Errors、重试与幂等

两层错误必须分开：HTTP/transport failure 与 HTTP 成功后的 generation outcome。OpenAI 除常规 4xx/429/5xx 外，Response 自身还能 `failed` 或 `incomplete`；Anthropic 的 error JSON 是可扩展 `{type,error:{type,message},request_id}`，并区分 429 rate limit、500 API、504 timeout、529 overloaded，且流中可能 200 后失败。[OpenAI error codes](https://developers.openai.com/api/docs/guides/error-codes) [Anthropic API errors](https://platform.claude.com/docs/en/api/errors)

截至截点，两家的 Messages/Responses create reference 都**没有定义可依赖的创建请求 idempotency key**。OpenAI 的 `X-Client-Request-Id` 只用于请求关联和支持排障，不是去重承诺。[OpenAI request IDs](https://developers.openai.com/api/reference/overview) Anthropic Messages reference 也只暴露创建/计数接口，没有单条 Message 的幂等创建资源。[Anthropic Messages reference](https://platform.claude.com/docs/en/api/messages/create) 所以：

- Yotta 为每次实际网络尝试生成新的 `AttemptID`，保留同一个逻辑 `RunID`；不得声称 provider exactly-once。
- 只在未消费任何输出、且失败明确可重试时自动新建 attempt；尊重 `Retry-After`。已经收到 delta、工具调用或遇到“请求可能已被接受但连接丢失”的超时，标成 `ambiguous`，不静默重放。
- provider SDK 的默认 retry 必须关闭或纳入 Yotta attempt journal；Anthropic 官方 SDK 默认会对连接错误、429 和 5xx 指数退避重试两次，否则日志中的一次调用实际可能是三次。[Anthropic API errors](https://platform.claude.com/docs/en/api/errors)
- 本地工具执行另设 durable ledger，键至少为 `(run_id, provider, provider_response_id, call_id)`；重复收到同一调用时返回已有结果，不再执行副作用。新的 provider attempt 产生新 call ID 时不能猜测它与旧调用等价，必须由 workflow 自己的 idempotency key/capability 控制。

## Cancellation 的真实强度

共享 `Cancel()` 只保证 Yotta 停止等待、停止发布后续事件和不再启动工具；provider 是否确认终止必须另报：

- OpenAI background Response 有 `/responses/{id}/cancel`，重复 cancel 是幂等的；同步 Response 的官方做法是终止连接。[OpenAI background mode](https://developers.openai.com/api/docs/guides/background)
- Anthropic 单条 Messages 没有服务端 cancel endpoint；SDK/context 或 stream abort 只能中止客户端连接，不能承诺服务端已停止或不计费。Message Batch 有独立 cancel API，但它是批处理资源，不是交互式 Message 的等价实现。[Anthropic Message Batches](https://platform.claude.com/docs/en/api/messages/batches)

建议记录 `Cancellation{local_stopped, provider_acknowledged, provider_status}`。只有 OpenAI background cancel 返回终态时才能把 `provider_acknowledged=true`；其他情况保持 false/unknown，不能伪造统一的强取消。

## Retention 不是一个通用 `store` 布尔值

OpenAI Responses 默认存储，`store:false` 可关闭供 API 后续检索的 application state；但默认 abuse-monitoring 日志仍可能保留最多 30 天。Responses application state 默认至少 30 天，ZDR 组织会强制 `store:false`；background 即使 `store:false` 也会为轮询在磁盘暂存约 10 分钟。[OpenAI data controls](https://developers.openai.com/api/docs/guides/your-data) Anthropic API 的标准策略是在 30 天内删除 inputs/outputs，但存在 Files、ZDR 合同、政策执行与法律例外；Messages 没有 per-request `store` 对应项。[Anthropic commercial retention](https://privacy.claude.com/en/articles/7996866-how-long-do-you-store-my-organization-s-data)

所以 `RetentionRequirement` 是 capability 要求，不是透传字段：

- `no_application_state`：OpenAI 强制 `store:false`，Anthropic 使用 stateless Messages；这**不等于 ZDR**。
- `zero_retention_required`：连接配置必须有经管理员确认的 ZDR entitlement，否则预检失败，绝不降级。Anthropic structured outputs 在 ZDR 下仍会把 schema grammar 技术缓存最多 24 小时，schema 中不得放 secret/PHI。[Anthropic structured-output retention](https://platform.claude.com/docs/en/build-with-claude/structured-outputs)
- provider-side conversations/`previous_response_id`/background 是显式 capability；默认 Yotta 3.1 应由自身保存最小 transcript，并选 `no_application_state`。

## 直接实施清单

1. 破坏性删除当前 `Chat/ChatStructured/ModeAuto/ModePrompt` 接口及 endpoint 猜测；新建 `Generate` + event stream + typed outcome。
2. OpenAI adapter 只实现 Responses；Anthropic adapter 只实现 Messages。Anthropic structured result 改用 `output_config.format`，不再强制 `result` tool。
3. 增加版本化 schema compiler、model capability preflight 和本地 validator；不支持就编译失败。
4. 增加 terminal reducer：覆盖 complete、tool calls、refusal、max/context truncation、content filter、pause、cancel、failed 与未知终态；SSE EOF 不是终态。
5. attempt journal 必须保存 Yotta AttemptID、provider request/response ID、resolved model、raw finish reason、usage snapshots、retry decision、cancel strength；error message 只用于展示，逻辑按 code/class 分支。
6. 工具执行采用先落 durable ledger、再执行副作用、最后落 result 的协议；parallel calls 全量处理，结果关联按 opaque call ID。
7. retention 在连接 capability 与 workflow requirement 两端声明并 fail closed；凭据、prompt、tool args/result、raw provider payload 默认不进普通日志。
8. conformance fixtures 至少覆盖：严格 JSON 成功/拒绝/截断/schema violation；单个与多个工具、工具错误、重复 call；429/5xx/529/200 后 stream error；本地 cancel 与 provider-ack cancel；request IDs；两家的 cache-token 口径；未知 event/type；`store:false` 与 ZDR requirement。

这套边界可以共享运行时和文档生成，又不会牺牲 provider-native 能力：公共契约稳定，wire transcript 与生命周期差异留在 adapter，任何不支持的语义都在预检期明确失败。
