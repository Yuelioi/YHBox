---
kind: note
summary: "Workflow 3.1 AI 节点、provider-native 生成契约、安装槽位、凭据和 consent 的当前边界"
activation: action
read_when: "改/排查 AI Generate/Extract 节点、结构化输出、模型 profile、provider adapter、AI settings/credential、workflow consent 或 AI resource session 前"
recheck_when: "增删 AI 节点 pin/config/capability; 改 GenerateRequest/Outcome/ProviderFailure; 改 provider wire API、profile digest、installation slot、credential binding、retention scope 或 consent identity"
---
# AI 节点与 provider-native runtime

Workflow 3.1 的 AI 单一实现路径是 `internal/ai` + `internal/nodes31` + `internal/nodes31runtime`。不存在通用 Chat 接口、endpoint 猜测、`ModeAuto/Native/Prompt` 或 JSON prompt/fence fallback；不要重新引入 `internal/services/llm`。

## 节点契约

- `AI Generate`：node type `https://schemas.yotta.dev/nodes/ai/generate/v1`，输入 `prompt:String`，输出 `result:String`。
- `AI Extract`：node type `https://schemas.yotta.dev/nodes/ai/extract/v1`，输入 `prompt:String`，输出 `result:JSON`；配置中的 `schema` 必须通过 `ai.StrictSchemaValidatorID`，运行时只接受 provider-native strict structured output。
- 两者配置共享 `slot`、`instructions`、`temperature`、`maxOutputTokens`；Extract 额外要求 `schema`。
- 两者都是 effectful、recorded、push、no-cache、retry-never、cooperative cancellation、required timeout；成功信号 `completed`，失败信号 `failed`。
- capability 是 `ai-generation`，target kind `ai-model`，credential required、sensitive、consent-once；runtime 只通过冻结的 requirement/session 调 provider，不从节点配置读取 endpoint、API key 或 provider 名称。

## Provider-native 边界

`internal/ai.Provider` 只有：

```go
Generate(context.Context, string, GenerateRequest) (Outcome, error)
```

- OpenAI 官方 adapter 只调用 Responses API；Anthropic adapter 只调用 Messages API。
- `GenerateRequest` 固定 attempt ID、prompt/instructions、limits、retention 和可选 strict `StructuredOutputSpec`。
- `Outcome` 保留 provider/model/request/response identity、typed output items、finish、token/cache/reasoning usage 和 cancellation；不能降格为纯文本字符串。
- `ProviderFailure` 的 stage/class/retry 用于控制流；message/raw 仅供诊断，普通日志必须脱敏。
- 结构化请求在 profile 未声明 capability 时立即失败；schema/返回值不精确匹配时是 contract failure，不做文本截取补救。

## Profile、安装与凭据

- settings 只保存 `AIModelSettings` 安装元数据：slot、label、provider、model、capabilities、token budget、evaluation 和 workflow consent。
- API key 只在 `AISecrets`，安全存储 target 是 `Yotta/AIModel/<slot>`；resource session 使用冻结的 `ai-credential/<slot>` binding 取值，节点和 workflow 看不到 secret。
- 启动时 `ai.Install` 将 settings sealing 为不可变 profile/installations。相同 profile digest 共享 native provider，但每个逻辑 slot 有独立 target `ai-model/<slot>`、credential binding 和 consent。
- profile/consent 以 canonical artifact digest 标识。profile 改动会令旧 workflow consent 失效；用户必须重新显式 grant。
- resource provider 只开放 `generate` / `generate-structured`；operation、structured flag 与 retention scope 必须和 capability grant 完全一致。

## 运行和观测

`internal/nodes31runtime/ai.go` 从 capability session 获得 sealed `Outcome`。Generate 只接受 text items；Extract 只接受一个 structured item；非 `completed` finish 走 `ai.generation_failed`。adapter trace 只记录安全的 provider/finish/request IDs 与 token counters，不记录 prompt、credential 或 provider raw body。

当前 AI 节点没有旧版动态 IO、vision/Image pin 或 provider model-list contract。需要这些能力时，应扩展 3.1 datatype/node contract 和 provider-native outcome，而不是恢复旧节点栈。
