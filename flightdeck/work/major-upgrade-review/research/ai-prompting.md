# Yotta 3.1：2026 模型时代的 Prompt、Tool 与 Agent 升级研究

> 调研日期：2026-07-13
> 来源范围：只采用 OpenAI、Anthropic 与 Model Context Protocol 的官方文档/规范；项目判断来自当前仓库源码。外部产品能力会继续变化，因此文中具体模型名用于说明“截至调研日的基线”，Yotta 的长期设计不应把滚动别名写死为协议。

## 结论先行

Yotta 的 AI 能力不应继续围绕“给任意端点发送两段字符串，再尽量把返回文本解析出来”扩建。3.1 应把 AI 子系统从一个薄 `Chat` adapter 升级为一个有明确模型能力、指令权限、结构化数据流、工具授权、会话状态、评测与追踪的深模块。

最关键的变化不是把旧提示词润色得更长，而是删除已经被新模型能力和运行时机制替代的文字脚手架：删除“逐步思考”、反复的 `CRITICAL/MUST`、只回 JSON 的提示词、Markdown 围栏剥离、端点猜测与结构化输出 fallback；把格式约束交给严格 schema，把工具边界交给类型与权限，把长会话交给状态/压缩，把模型升级交给 eval gate。

截至调研日，OpenAI 官方模型指南以 GPT-5.6 为当前系列，并明确建议 reasoning、tool calling 与多轮工作流使用 Responses API；同时提醒更聪明的模型能从上下文推断意图，不必规定每一步，但仍要提供领域上下文、硬约束、审批边界和成功标准。[OpenAI：Model guidance](https://developers.openai.com/api/docs/guides/latest-model)

这意味着 Yotta 3.1 应采用以下总原则：

1. **模型是经评测的运行配置，不是节点里的自由字符串。** 默认使用固定 snapshot；滚动 alias 只能是用户明确选择的实验策略。OpenAI 官方同样建议生产应用固定 model snapshot，并用测试/eval 监控升级。[OpenAI：Text generation](https://developers.openai.com/api/docs/guides/text)
2. **Prompt 是代码与 typed builder，不是散落字符串。** 产品内置 prompt 应与功能同目录、接受 typed inputs、走代码评审和测试；用户自定义 prompt 则是有 schema/version/hash 的工作流资产。OpenAI 已宣布弃用 API reusable prompt objects，并建议将生产 prompt 存在代码中。[OpenAI：Prompt engineering](https://developers.openai.com/api/docs/guides/prompt-engineering)
3. **结构化输出和工具调用必须 schema-first、strict、fail closed。** OpenAI 建议能用 Structured Outputs 时不要用 JSON mode，并建议函数调用始终开启 strict；Anthropic 也提供 strict tool use。[OpenAI：Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs)；[OpenAI：Function calling](https://developers.openai.com/api/docs/guides/function-calling)；[Anthropic：Tool use](https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview)
4. **Prompt injection 是数据流与权限问题，不是再加一段“忽略恶意指令”就能解决。** 不可信内容不得进入高权限指令；敏感工具必须最小权限、显式审批、可审计。[OpenAI：Safety in building agents](https://developers.openai.com/api/docs/guides/agent-builder-safety)；[Anthropic：Mitigate jailbreaks and prompt injections](https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/mitigate-jailbreaks)
5. **没有 trace 与 eval 的 prompt 不能发布。** OpenAI 将 trace grading 用于定位工具选择、handoff、指令/安全违规和 prompt/routing 回归，并建议稳定后转为可重复 dataset/eval run。[OpenAI：Evaluate agent workflows](https://developers.openai.com/api/docs/guides/agent-evals)

## 当前 Yotta 的具体缺口

以下不是抽象最佳实践，而是当前源码与 2026 官方基线之间的差距。

| 优先级 | 当前事实 | 3.1 问题 | 应采取的破坏性变化 |
| --- | --- | --- | --- |
| P0 | `internal/services/llm/openai.go:46,85` 仍调用 Chat Completions | 丢失 Responses 的 reasoning items、state、compaction、原生 agentic loop 与更佳缓存利用 | OpenAI 官方 adapter 改为 Responses-only；删除官方 OpenAI 的 Chat Completions 路径 |
| P0 | `provider.go:12-14` 只有 `system/user/assistant`；OpenAI 映射仍用 `SystemMessage` | 没有显式 developer authority；内部语义被旧 API 角色绑死 | 内部改为 `Instructions + InputBlock`；OpenAI 映射 `instructions/developer`，Anthropic 映射顶层 `system` |
| P0 | `ai.go:83` 允许动态 `{{value}}` 插入 System；`template.go:28` 直接 `fmt.Sprintf` | 不可信工作流输入可被提升到最高指令层，形成 prompt injection 高危路径 | Instructions 默认只允许静态 literal/受信 PromptRef；动态值只能进入 typed user/context/tool-result block |
| P0 | `ModePrompt`、`structuredViaPrompt`、JSON 围栏/首尾花括号容错解析仍存在 | 结构化输出只是“希望模型听话”，且会把错误文本误判为数据 | 删除 `ModePrompt`、`auto → prompt` 和解析兜底；端点不支持 strict native schema 时编译失败 |
| P0 | `SchemaField` 只有 name/type；AI 节点缺字段时 `continue` 并保留旧变量值 | 不是完整 JSON Schema；“成功但数据不完整”会污染自动化状态 | 支持嵌套、enum、description、required/nullable、范围/长度；缺必填字段整次调用失败，不保留旧值 |
| P1 | `ChatResponse` 只有 `Text` | finish/stop reason、usage、cache、reasoning、request ID、结构化 items 全丢失 | 返回 typed `ModelResult`，保留 output items、usage、stop reason、provider metadata 与 trace context |
| P1 | 节点直接暴露 `Model/Temperature/MaxTokens`，Anthropic 未填时硬补 1024 | 参数不是按模型能力校验；截断可能被当正常成功；“0”同时表示显式值和未设置 | 节点引用 `ModelProfile`；profile 明确 effort、输出预算、timeout、retention、cache，unsupported 参数在 compile 阶段拒绝 |
| P1 | `isOfficialEndpoint` 用 URL substring 决定 native/fallback | 能力由域名猜测，既脆弱也无法表达兼容端点的真实差异 | 每个 provider/endpoint 有显式 capability manifest；禁止运行时试错降级 |
| P1 | 没有 prompt ID/version/hash、模型 snapshot、toolset hash 的统一追踪 | 无法重放“哪版 prompt + 哪个模型 + 哪套工具”产生的结果 | PromptManifest 与 ModelRun trace 成为正式 contract |
| P1 | 没有仓库级 prompt/tool eval gate | 模型或提示词升级只能凭感觉验收 | 增加 fixtures、trace grading、离线 contract tests、在线 canary eval 和发布阈值 |
| P0 | `main.go:339-340` 每次启动固定监听 `127.0.0.1:8765/mcp`；`list_windows` 不受 armed 闸 | loopback 不是身份验证；读取窗口标题/进程也是敏感能力 | 默认不启动；优先本地 stdio；若启用 HTTP，则临时端口 + session credential + capability approvals |
| P1 | MCP 工具几乎都返回大段 text；`run_node.params` 是无约束 object | 无 output schema、低信号大输出、难验证；任意节点参数扩大攻击面 | MCP 2025-11-25 typed input/output schema、客户端/服务端双校验；按 namespace/capability 延迟发现工具 |

OpenAI 已明确推荐所有新项目使用 Responses API，并称其为统一的 agent-like interface，支持内建工具、多轮状态、reasoning/tool context 与结构化 Items。[OpenAI：Migrate to Responses](https://developers.openai.com/api/docs/guides/migrate-to-responses) 因此，对“不需要兼容、不需要兜底”的 Yotta 3.1，继续保留官方 OpenAI Chat Completions 没有收益。

## 模型选择与升级策略

### 不要再让节点直接保存任意模型字符串

当前 `AI` 节点保存 connection + model 字符串。这会让同一个 workflow 在模型 alias 漂移后无声变行为，也无法在编译时判断 vision、strict schema、tools、reasoning、context persistence 是否可用。

建议引入不可变 `ModelProfile`：

```text
ModelProfile
  id
  providerKind          openai-responses | anthropic-messages | explicit-plugin
  modelSnapshot         固定版本；rolling alias 必须显式 opt-in
  capabilities          vision, strictOutput, strictTools, toolSearch, reasoning, state
  reasoningPolicy       none/low/medium/high…，按 provider 映射
  outputBudget
  timeout
  cachePolicy
  retentionPolicy       stateful | stateless | ZDR-compatible
  safetyPolicyRef
```

OpenAI 官方建议生产使用固定 snapshot，并通过 eval 决定版本升级；其当前 GPT-5.6 迁移指南还建议从现有 reasoning effort 开始，对同档和低一档做代表性任务比较，而不是假设越高越好。[OpenAI：Text generation](https://developers.openai.com/api/docs/guides/text)；[OpenAI：Model guidance](https://developers.openai.com/api/docs/guides/latest-model)

因此，Yotta 的升级流程应是：注册候选 snapshot → 在固定 eval corpus 上比较质量/工具正确率/安全/延迟/成本 → 达标后新增 profile 版本 → workflow 明确升级引用。绝不能在原 profile 下静默替换 model string。

### 不做“最低公分母 Provider”

OpenAI Responses 和 Anthropic Messages 的状态、reasoning、tool/caching 语义不同。一个只暴露 `Chat(Text) Text` 的接口看似跨厂商，实际会抹掉最有价值的能力。

建议共享的是 Yotta 的领域语义，而不是供应商 API 的最低公分母：

```go
type ModelRequest struct {
    Profile      ModelProfileRef
    Instructions PromptRef
    Input         []InputBlock
    Output        *SchemaRef
    Toolset       *ToolsetRef
    Context       ContextPolicy
    Budget        RunBudget
}

type ModelResult struct {
    Items      []OutputItem
    Structured JSONValue
    Usage      Usage
    Stop       StopReason
    Provider   ProviderMetadata
    Trace      TraceContext
}
```

OpenAI adapter 可以使用 Responses state/reasoning items；Anthropic adapter 使用 Messages、顶层 system 和自己的 cache controls。公共层只承诺 Yotta 能验证的领域 contract；某 profile 缺少必需 capability 时，workflow compiler 直接报错。

### 第三方“OpenAI 兼容”必须降级为显式插件能力，不得伪装官方语义

3.1 core 可只内置官方 `openai-responses` 与 `anthropic-messages`。如仍要支持 Ollama、LM Studio 或其他网关，应作为明确 provider/plugin，声明自己支持的 capability；不能再由 BaseURL 域名判断，也不能请求失败后从 native 自动切到 prompt parsing。

这不是删除本地模型生态，而是删除“兼容接口等于兼容行为”的错误承诺。

## Developer/System 指令与 Prompt 结构

### 权限层必须在数据模型里可见

OpenAI 将 developer message 定义为应用开发者提供、优先于 user message 的规则与业务逻辑，并把 developer/user 类比为“函数定义/函数参数”。[OpenAI：Text generation](https://developers.openai.com/api/docs/guides/text) OpenAI 也明确警告：不可信变量若直接插入 developer message，攻击者将获得最高程度的控制；不可信输入应放在 user message。[OpenAI：Safety in building agents](https://developers.openai.com/api/docs/guides/agent-builder-safety)

Yotta 应把一个调用拆成：

- `Instructions`：产品/工作流作者的静态规则、目标、审批边界；不可从运行时动态 pin 插值。
- `UserInput`：本次用户请求和可信度未知的动态变量。
- `ContextBlock`：文件、截图 OCR、网页、工具结果等外部数据，携带 `source`、`trust`、`mime`、`sensitivity` 元数据。
- `Examples`：与 PromptRef 一同版本化的少量代表性示例，而不是每次由节点自由拼接。

Anthropic 同样建议把第三方内容放在 `tool_result`，而不是 system 或普通文本中；明确标注来源、JSON 编码不可信字符串、限制敏感数据/动作权限，并对工具结果做 injection screening。[Anthropic：Mitigate jailbreaks and prompt injections](https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/mitigate-jailbreaks)

当前允许 `{{dynamic}}` 同时进入 System 和 User 的设计必须被 breaking 删除。若确有动态高权限策略，只能来自受信、typed、经校验的 enum/配置对象，不能来自任意字符串。

### 内置 Prompt 的推荐最小结构

OpenAI 建议 developer prompt 常见顺序为 Identity、Instructions、Examples、Context，并使用 Markdown/XML 表达边界。[OpenAI：Prompt engineering](https://developers.openai.com/api/docs/guides/prompt-engineering) Anthropic 也建议清晰直接、给必要上下文、用相关且多样的示例、用 XML 区分 instructions/context/input。[Anthropic：Prompting best practices](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices)

Yotta 的内置 prompt builder 可采用：

```text
# Goal
一句话任务与完成标准

# Rules
只保留模型无法从 schema/code 推导的业务规则与硬边界

# Tool policy
允许的动作、何时调用、何时需要审批、停止条件

# Examples
仅在 eval 证明有收益时加入 1–少量高信号例子

# Context
typed blocks；动态且不可信，永远在稳定前缀之后
```

Prompt 长度不是质量指标。Reasoning 模型官方指导是“简单、直接”，且不需要要求它输出或解释 chain-of-thought。[OpenAI：Reasoning best practices](https://developers.openai.com/api/docs/guides/reasoning-best-practices)

## 哪些旧式提示词应删除

| 删除对象 | 为什么现在应删除 | 替代机制 |
| --- | --- | --- |
| “Think step by step / 展示完整思维链 / 先逐步推理” | reasoning 模型内部完成推理，OpenAI 明确称此类提示不必要 | 写目标、成功标准和可验证条件；需要质量时调 reasoning policy，并让模型在结束前核对结果 |
| 为防模型偷懒而堆叠 `CRITICAL`、`MUST`、全大写、重复三遍 | Anthropic 指出较新模型对 system 更敏感，旧 anti-undertrigger prompt 会造成工具 overtrigger，建议改回普通表述 | 一条清晰触发规则 + tool_choice/capability policy + eval |
| “只返回 JSON、不要 prose、不要代码围栏”以及围栏/首尾 `{}` 容错解析 | JSON mode 只保证合法 JSON，不保证 schema；自由文本解析会掩盖失败 | strict Structured Outputs / strict tool schema；schema 不支持则 fail closed |
| 用 assistant prefill 强迫格式、去前言或维持角色 | Anthropic 4.6+ 已不支持最后一个 assistant prefill，并说明新模型多数场景不再需要 | system/instructions + structured outputs |
| 把所有动态数据拼进 system/developer | 将不可信输入提升到高权限，是直接 injection 面 | user/context/tool-result typed blocks；trust label 与 policy engine |
| 把全部工具 schema 每轮都塞进 prompt | 工具定义占 context/token；工具越多选择越难 | 小型初始工具集；namespace + deferred tool search；按任务编译 toolset |
| 用 prompt 描述代码已经知道的参数、顺序和确定性逻辑 | 增加模型出错机会和 token；OpenAI 建议已知参数由代码填、固定连续操作合并 | typed tool adapter、代码编排或 bounded programmatic orchestration |
| 远端 prompt object 作为生产真相 | 难以与源码、schema 和 release 原子版本化；OpenAI 正在弃用该 API | code-owned PromptManifest；用户 prompt 作为 workflow asset |
| “若 native 失败就改用 prompt JSON” | 失败后换语义会让同一 workflow 非确定地改变 contract | capability compile check；unsupported 就停止 |
| 一个跨厂商通用 prompt 试图修正所有模型差异 | provider/model snapshot 的 tool、reasoning、cache 行为不同 | 公共业务 prompt + 小型、显式、经 eval 的 provider overlay |
| 无界的“尽可能多搜索/一直尝试/不要停” | 新模型更主动，容易扩大成本、延迟和副作用 | `RunBudget`：max turns、tool calls、tokens、deadline、retry、stop condition |

对应的一手依据：[OpenAI reasoning prompting](https://developers.openai.com/api/docs/guides/reasoning-best-practices)、[Anthropic modern prompting and migration](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices)、[OpenAI Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs)、[OpenAI function calling](https://developers.openai.com/api/docs/guides/function-calling)、[OpenAI code-managed prompts](https://developers.openai.com/api/docs/guides/prompt-engineering)。

## 哪些约束仍必须保留

模型变聪明不等于可以删除业务边界。OpenAI 当前模型指南明确说可以少规定步骤，但仍应提供领域上下文、硬约束、审批边界和成功标准。[OpenAI：Model guidance](https://developers.openai.com/api/docs/guides/latest-model)

以下约束必须保留，而且应尽量从 prose 上移到机器可执行层：

- **目标与完成标准**：输出要解决什么、怎样算完成、必须验证什么。
- **领域事实与当前上下文**：训练数据不可能知道的 Yotta schema、当前窗口/工作流/资产状态。
- **权限和审批边界**：文件、网络、进程、输入注入、截图、保存/删除、MCP 外发等；这是 policy engine，不只是 prompt。
- **工具语义**：工具做什么、何时用/不用、参数含义、返回范围、是否有副作用；复杂工具需要 provider 适配后的清晰描述。
- **停止与预算**：最大回合、调用次数、deadline、重试次数、重复调用禁止、部分失败策略。
- **输出 contract**：字段、类型、枚举、必填、nullable、单位；放 JSON Schema，不靠自然语言重复。
- **不可信数据规则**：工具结果、网页、OCR、上传文件中的文字只是数据，不能覆盖指令。
- **重要歧义策略**：哪些歧义必须询问，哪些可以在明确默认下继续。
- **高风险人工复核**：不可逆或高影响动作必须确认；OpenAI 和 MCP 官方均强调 human-in-the-loop。[OpenAI：Safety best practices](https://developers.openai.com/api/docs/guides/safety-best-practices)；[MCP：Tools 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/server/tools)

一个简单判断标准：如果约束能由类型、编译器、权限、事务或代码确定性执行，就不要只写在 prompt；如果它是模型完成语义任务所需的意图/上下文，才留在 prompt。

## Structured Outputs 与 Tool Schema

### 删除扁平 `SchemaField`

OpenAI strict function calling 要求每个 object 的 `additionalProperties: false`，properties 全部列为 required；可选值用包含 `null` 的类型表达。官方建议始终开启 strict。[OpenAI：Function calling](https://developers.openai.com/api/docs/guides/function-calling)

Yotta 的 `JSONSchema{Fields []SchemaField{Name, JSONType}}` 应由真正的 JSON Schema contract 替换，至少支持：

- object/array 递归结构；
- `enum` / discriminated union；
- description/title；
- required 与 nullable 分离；
- number/string/array constraints；
- `additionalProperties: false`；
- stable schema ID + content hash；
- provider 支持子集的 compile-time validation。

Structured Outputs 应用于模型给工作流返回 typed data；function/tool calling 应用于模型请求 Yotta 执行动作。OpenAI 官方明确区分了这两个场景。[OpenAI：Structured Outputs](https://developers.openai.com/api/docs/guides/structured-outputs)

### 工具不是函数名列表，而是面向模型的 API 产品

OpenAI 建议函数明显直观、用 enum/object 让非法状态不可表示、把代码已知参数从模型参数中移除、合并固定顺序的调用，并让初始可见工具尽量少（软建议少于 20）。[OpenAI：Function calling](https://developers.openai.com/api/docs/guides/function-calling)

Anthropic 强调工具描述是工具性能最重要的因素：应解释做什么、何时用/不用、各参数语义和限制；复杂输入可给 schema-validated examples，并建议返回高信号字段、减少无关大响应。[Anthropic：Define tools](https://platform.claude.com/docs/en/agents-and-tools/tool-use/define-tools)

两家指导并不矛盾：**初始工具面要小、schema 要精确；被选中的复杂工具描述要足够完整。** Yotta 应允许 provider adapter 对同一领域工具生成不同密度的描述，而不是把所有 provider 强压成同一段文本。

建议工具 contract 包含：

```text
ToolDefinition
  canonicalName / namespace
  summary                 供工具发现
  description             加载后使用说明
  inputSchema / outputSchema
  sideEffect              none | read-sensitive | reversible-write | irreversible
  capabilities            filesystem/network/process/input/capture/llm/...
  approvalPolicy
  idempotencyPolicy
  timeout / retryPolicy
  resultSensitivity
```

其中 `sideEffect`、`idempotencyPolicy` 是基于官方审批/安全要求作出的 Yotta 领域设计推论，不应交给模型自行判断。

### 大节点目录必须按需发现

OpenAI tool search 可以按需把工具加载进 context，避免每轮放入全部 definitions；官方建议用 namespace/MCP server，namespace 以少于 10 个函数为性能目标，并把丰富细节留到实际加载后的函数描述。[OpenAI：Tool search](https://developers.openai.com/api/docs/guides/tools-tool-search)

Yotta 节点 catalog 很大，因此 MCP/Agent 不应先调用一个返回全量大 JSON 的 `list_nodes`。建议改为：

- `yotta.catalog.search(query, category, capability)`：只回候选摘要；
- `yotta.catalog.describe(kind)`：按需回完整 schema；
- `yotta.workflow.validate(graph)`：严格 typed diagnostics；
- `yotta.workflow.save(graph)`：write approval；
- `yotta.inspect.windows(...)`：敏感 read approval；
- `yotta.execute.<capability>`：按 capability namespace 延迟加载，不用一个 `run_node(kind, params:any)` 穿透全部节点。

如 provider 不支持原生 tool search，Yotta 自己的 `ToolResolver` 也应先根据 workflow/capability 编译一个小 toolset，而不是退回“全部注入”。

### Tool output 也必须有 schema

MCP 2025-11-25 允许工具声明 `outputSchema`；server 必须返回符合 schema 的 structured result，client 应再次验证。规范还要求 server 验证输入、实施访问控制与限流、清理输出；client 应对敏感操作确认、调用前展示输入、验证结果、设置 timeout 并记录审计日志。[MCP：Tools 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/server/tools)

因此 Yotta MCP 不应再以 `ToolResultText(string(json))` 为主。结构化字段放 `structuredContent`，给人看的摘要放 text；两边都通过 schema。大图像、日志、catalog 用 resource link/分页，不塞入一段无限文本。

## Prompt Caching 与 Context Management

### 稳定前缀、动态后缀

OpenAI 缓存要求 exact prefix match，建议把静态 instructions/examples 放在开头，把用户变量放在结尾；tools、images 和 structured schema 也会参与缓存。[OpenAI：Prompt caching](https://developers.openai.com/api/docs/guides/prompt-caching) Anthropic 同样建议静态 tool definitions、system instructions、context、examples 在前，并明确其缓存层级是 `tools → system → messages`。[Anthropic：Prompt caching](https://platform.claude.com/docs/en/build-with-claude/prompt-caching)

Yotta PromptBuilder 的确定性顺序应固定为：

1. toolset/version；
2. product instructions + prompt version；
3. stable examples/reference context；
4. prior conversation state；
5. current dynamic input。

不要在稳定 system 前缀中放时间戳、随机 ID、运行时变量，也不要无意义地重排工具。记录 `cached_tokens/cache_write_tokens`（OpenAI）或 `cache_read/cache_creation`（Anthropic），否则无法知道优化是否真实生效。[OpenAI：Prompt caching](https://developers.openai.com/api/docs/guides/prompt-caching)；[Anthropic：Prompt caching](https://platform.claude.com/docs/en/build-with-claude/prompt-caching)

### 长会话不能只靠无限追加 messages

OpenAI Responses 可使用 `previous_response_id`/state，且提供 server-side 或 standalone compaction；compaction item 是不透明的 canonical continuation state，不应由业务代码拆解或重新总结。[OpenAI：Compaction](https://developers.openai.com/api/docs/guides/compaction)

Yotta 应提供明确的 `ContextPolicy`：

- `single_turn`：纯生成/提取节点；
- `run_scoped`：同一次 workflow execution 内共享状态，结束即清除；
- `conversation`：显式 session，持久化 provider continuation token/item；
- `stateless_compacted`：本地保存 canonical compacted items；
- `sensitive`：`store=false`，按 provider 能力处理 encrypted reasoning/compaction。

不得让普通 AI 节点隐式共享全局聊天历史；状态必须有 owner、TTL、数据保留说明和清除入口。

## Prompt Versioning、Evals 与 Trace

### 两类 Prompt、两个 owner

1. **Yotta 内置 Prompt**：在源码中，靠代码 review、测试、release version 管理。OpenAI 当前也建议 code-managed prompts、typed inputs、fixtures/evals 和正常部署流程。[OpenAI：Prompt engineering](https://developers.openai.com/api/docs/guides/prompt-engineering)
2. **用户 Prompt Asset**：属于 v3 workspace schema；保存 `schemaVersion`、内容、输入/输出 contract、创建来源和不可变 hash。workflow 引用具体 revision，而不是可变名称。

建议 `PromptManifest`：

```text
id / revision / sha256
owner
purpose
instructionsTemplate
inputSchema / outputSchema
compatibleCapabilities
providerOverlays
toolsetRef
evalSuiteRef
securityPolicyRef
```

一次 model run 的 trace 至少记录：model profile + snapshot、prompt revision/hash、input/output schema hash、toolset hash、reasoning/effort、stop reason、tool calls/approvals、token/cache usage、latency、provider request ID、Yotta workflow/run/node ID。

### Eval 必须进入 CI 与升级流程

OpenAI 建议 eval-driven development、task-specific 数据、完整日志、自动评分、持续评测；反对“看起来不错”的 vibe-based eval。[OpenAI：Evaluation best practices](https://developers.openai.com/api/docs/guides/evaluation-best-practices) Anthropic 同样要求先定义 specific/measurable success criteria，测试真实分布与 edge cases，优先可自动评分的方法。[Anthropic：Define success criteria and build evaluations](https://platform.claude.com/docs/en/test-and-evaluate/develop-tests)

Yotta AI eval corpus 应覆盖：

- 正常、边界、对抗、多语言输入；
- vision 截图、OCR 中的间接 injection；
- structured output 完整性和 schema 拒绝；
- 工具选择、参数精度、重复调用、停止条件；
- 高风险动作必须审批、越权工具不可见；
- tool error、timeout、partial result、context compaction；
- 成本、延迟、cache hit、输出 token budget；
- 相同 workflow 在候选 model snapshot 上的回归比较。

评分优先级：代码/精确 contract → pass/fail 或 pairwise model grader → 经校准的人审。OpenAI 指出 LLM 在比较/分类上通常比开放式生成评分可靠，且 model grader 必须与人类标签校准。[OpenAI：Evaluation best practices](https://developers.openai.com/api/docs/guides/evaluation-best-practices)

建议门禁：

- PR：prompt builder/schema/tool contract 的离线 golden + property tests；
- 受信 secret 环境的 scheduled/labelled job：固定 snapshot 在线 eval；
- 模型升级 PR：旧/新 profile 成对比较报告；
- release：关键安全、tool selection、schema validity 均不得回退，成本/延迟超过预算则阻断。

### Trace 是一等产物，但必须脱敏

OpenAI Agents tracing 会记录 model calls、tool calls/output、handoff、guardrail 和 custom spans；trace grading 再用结构化标准找 workflow 级失败。[OpenAI：Integrations and observability](https://developers.openai.com/api/docs/guides/agents/integrations-observability)；[OpenAI：Evaluate agent workflows](https://developers.openai.com/api/docs/guides/agent-evals)

这也意味着 trace 可能包含敏感数据。Yotta 应在写 trace 前按 schema 标记与脱敏 secret、token、截图/窗口标题、文件内容和 PII；MCP 官方安全指南同样明确禁止记录 credentials，并要求敏感结构化日志字段脱敏。[MCP：Authorization security tutorial](https://modelcontextprotocol.io/docs/tutorials/security/authorization)

## Agent Safety 与 Prompt Injection

### 权限不能只写在 Prompt 里

OpenAI 明确说 agents 仍会犯错或被欺骗，应谨慎授予访问；建议不可信数据不进 developer message、节点间用 structured outputs、MCP 保留审批、输入 guardrail、trace graders/evals 多层组合。[OpenAI：Safety in building agents](https://developers.openai.com/api/docs/guides/agent-builder-safety)

Yotta 应把 `flightdeck/design.md` 已提出的 permission manifest 延伸到 AI/Agent：

```text
filesystem roots
network hosts + redirect/DNS policy
process/shell
window enumeration
screen capture / OCR
mouse/keyboard input
workflow save/delete/run
MCP data egress
LLM provider + data retention
```

Model 只能在编译后允许的 toolset 内选择；permission engine 在模型之外再次校验每次调用。审批绑定规范化后的具体参数与 capability，参数改变则原批准失效。

### Prompt injection 防御层

1. **Authority isolation**：不可信数据永不进入 Instructions/developer/system。
2. **Typed boundaries**：节点间、tool input/output 使用 strict schema，拒绝额外字段。
3. **Least privilege**：按任务编译最小 toolset；不把秘密放进不需要它的上下文。
4. **Approval**：敏感读、写、执行、外发均有明确 UI；不是只有“写”才危险。
5. **Sandbox/policy**：本地 MCP、script、shell、网络和文件受 OS/应用 policy 限制。
6. **Screening**：对 user input 与外部 tool result 可增加轻量 injection classifier，但它只是 defense-in-depth。
7. **Adversarial eval**：把恶意网页、邮件、OCR、tool output 放入持续回归集。
8. **Audit and response**：记录经过脱敏的决策、审批与调用，支持撤销/报告。

Anthropic 官方将 direct 与 indirect prompt injection 分开建模，并建议对工具输出做来源标注、JSON 编码、最小权限、sandbox、screening 与 red-team。[Anthropic：Mitigate jailbreaks and prompt injections](https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/mitigate-jailbreaks) OpenAI 也建议 adversarial testing、HITL 和限制开放输入/输出范围。[OpenAI：Safety best practices](https://developers.openai.com/api/docs/guides/safety-best-practices)

### MCP 的具体 breaking 方案

MCP 2025-11-25 是调研日官方标为 latest 的稳定规范。[MCP：Tools 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/server/tools) 对 Yotta 本地桌面应用：

- 默认不启动 MCP；
- 首选 stdio，让连接生命周期属于显式启动它的 client；MCP 授权规范也区分 HTTP 与 stdio，HTTP 走授权框架，stdio 从环境获取 credentials。[MCP：Authorization 2025-11-25](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)
- 若保留 loopback HTTP：随机 session credential、临时/可配端口、origin/host 校验、启动 UI 指示、闲置关闭；不得把 `127.0.0.1` 当 auth。
- `armed` 不能代替身份与逐能力批准；`list_windows`、截图、读 graph 同样可能泄露敏感数据。
- 所有 tools 有 input/output schema、timeout、rate limit、audit；敏感 tool 输入在确认 UI 中完整展示。
- 本地 server 以最低 OS 权限运行，文件/网络/进程访问受 manifest/sandbox。MCP 官方安全最佳实践也建议本地 server 最小默认权限、限制 filesystem/network、显式增权，HTTP transport 必须限制未授权进程访问。[MCP：Security best practices](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices)

## 建议的 Yotta 3.1 AI 模块

```text
internal/ai
  modelpolicy/       profile、snapshot、capability、budget、retention
  prompt/            PromptManifest、typed builder、provider overlay
  schema/            JSON Schema subset + provider compiler
  tools/             canonical tools、resolver、approval metadata
  session/           state owner、compaction、TTL、clear
  eval/              fixtures、graders、comparison report
  trace/             spans、usage、redaction、export
  safety/            trust labels、injection screening、policy bridge
  provider/
    openairesponses/ Responses API 专用 adapter
    anthropic/       Messages API 专用 adapter
```

节点层不应直接依赖供应商：

- `AI Generate`：单轮文本/多模态生成；可选严格 output schema；无工具。
- `AI Extract/Classify`：必须有 strict schema；默认低/无 reasoning profile；缺字段整体失败。
- `AI Agent`：显式 ToolsetRef、PermissionPolicyRef、RunBudget、ContextPolicy；拥有可观察 agent loop。
- `AI Session`（若产品确实需要）：显式创建/继续/关闭 conversation state；普通节点不能暗中共享。

`System` pin 改名为 `Instructions`，默认是静态配置，不允许动态 edge。`User` 改为 typed input blocks。输出不再把结构化字段附在一个可变 map 上，而是 `Text` 与 `Structured` 两种互斥/明确的 result contract。

## 实施顺序

### AI Phase 0 — 冻结语义并建立评测

1. 收集当前 AI 节点的代表性 workflows、vision、structured output 和失败案例。
2. 为现行为建立 eval corpus，但不把 prompt fallback 等旧实现当正确 contract。
3. 定义质量、安全、schema、tool、成本和延迟阈值。

验收：模型或 prompt 变更能输出可比较报告；不再靠手工聊天判断。

### AI Phase 1 — 新 v3 contract，一次删除旧 fallback

1. 引入 ModelProfile、PromptManifest、InputBlock、完整 SchemaRef、ModelResult。
2. 删除 `ModePrompt`、`ModeAuto`、`structuredViaPrompt`、围栏/花括号解析、endpoint substring capability 判断。
3. 删除缺字段 `continue`、Anthropic 1024 隐式默认与 `Temperature > 0` 这种歧义参数编码。

验收：unsupported capability 在 compile 阶段报 typed diagnostic；运行中绝不换协议/语义。

### AI Phase 2 — Provider-native adapters

1. OpenAI 改 Responses-only，支持 instructions/developer、Items、strict output、usage/stop、reasoning、state/cache。
2. Anthropic 使用顶层 system、strict tool/output、provider cache/context 特性。
3. provider contract tests 固定实际 wire request/response，并覆盖 refusal、truncation、timeout、schema reject。

验收：官方 provider 不走 Chat Completions；所有非正常 stop reason 显式处理。

### AI Phase 3 — Prompt 与节点产品化

1. 内置 prompt 移到 feature-local typed builder，并有 ID/revision/hash。
2. Instructions 禁止动态不可信插值；用户/外部数据进入带 trust/source 的 blocks。
3. 拆分 Generate、Extract/Classify、Agent，避免一个节点靠 mode 字符串切换完全不同语义。

验收：仓库内无“ONLY JSON”式格式提示；prompt snapshot 与 schema snapshot 可重放。

### AI Phase 4 — Tool/MCP capability model

1. Canonical ToolDefinition + strict input/output + side effect/approval/capability metadata。
2. namespace/resolver 延迟加载；删除全量 catalog blob 和 `params:any` 的万能执行入口。
3. MCP 默认关闭；stdio-first 或 authenticated loopback；permission manifest 每次执行再校验。

验收：未授权模型看不到/调用不了危险工具；所有敏感调用可拒绝、可审计。

### AI Phase 5 — Context、cache、trace、持续 eval

1. 稳定 prompt prefix 与 cache metrics。
2. session owner/TTL/compaction/clear UX。
3. 全链 trace + redaction；CI/canary eval gate；model profile 升级报告。

验收：任一生产结果可回答“哪个模型 snapshot、哪版 prompt/schema/toolset、用了什么权限、为何停止、成本/延迟多少”。

## 完成定义

- 官方 OpenAI 只使用 Responses API；没有 Chat Completions compatibility path。
- 没有 `ModePrompt`、JSON prose injection、围栏/花括号容错解析或 native→prompt fallback。
- Instructions/developer/system 中不存在运行时不可信字符串插值。
- 所有结构化输出和工具调用使用 strict schema；缺字段、额外字段、错误 stop reason 均失败。
- ModelProfile 固定 snapshot 和 capabilities；升级必须通过 eval comparison。
- PromptManifest、schema、toolset 均有 revision/hash，trace 可重放来源。
- Agent 有 max turns/tool calls/tokens/deadline/retry/stop condition，不存在无界自主循环。
- 工具初始面最小、支持 namespace/deferred discovery；input/output 双向验证。
- MCP 默认不监听；启用时有身份、最小权限、逐能力审批、timeout/rate limit/audit。
- prompt/tool/model 变更在 CI 或受控 canary 中跑正常、边界、对抗与安全 eval。

## 一手来源索引

### OpenAI

- 当前模型与迁移基线：[https://developers.openai.com/api/docs/guides/latest-model](https://developers.openai.com/api/docs/guides/latest-model)
- Text、role、snapshot 与 code-managed prompt：[https://developers.openai.com/api/docs/guides/text](https://developers.openai.com/api/docs/guides/text)
- Prompt 结构与版本管理：[https://developers.openai.com/api/docs/guides/prompt-engineering](https://developers.openai.com/api/docs/guides/prompt-engineering)
- Reasoning prompt：[https://developers.openai.com/api/docs/guides/reasoning-best-practices](https://developers.openai.com/api/docs/guides/reasoning-best-practices)
- Responses 迁移：[https://developers.openai.com/api/docs/guides/migrate-to-responses](https://developers.openai.com/api/docs/guides/migrate-to-responses)
- Structured Outputs：[https://developers.openai.com/api/docs/guides/structured-outputs](https://developers.openai.com/api/docs/guides/structured-outputs)
- Function calling：[https://developers.openai.com/api/docs/guides/function-calling](https://developers.openai.com/api/docs/guides/function-calling)
- Tool search：[https://developers.openai.com/api/docs/guides/tools-tool-search](https://developers.openai.com/api/docs/guides/tools-tool-search)
- Programmatic Tool Calling：[https://developers.openai.com/api/docs/guides/tools-programmatic-tool-calling](https://developers.openai.com/api/docs/guides/tools-programmatic-tool-calling)
- Prompt caching：[https://developers.openai.com/api/docs/guides/prompt-caching](https://developers.openai.com/api/docs/guides/prompt-caching)
- Context compaction：[https://developers.openai.com/api/docs/guides/compaction](https://developers.openai.com/api/docs/guides/compaction)
- Agent evals / trace grading：[https://developers.openai.com/api/docs/guides/agent-evals](https://developers.openai.com/api/docs/guides/agent-evals)
- Evaluation best practices：[https://developers.openai.com/api/docs/guides/evaluation-best-practices](https://developers.openai.com/api/docs/guides/evaluation-best-practices)
- Agents observability：[https://developers.openai.com/api/docs/guides/agents/integrations-observability](https://developers.openai.com/api/docs/guides/agents/integrations-observability)
- Agent prompt-injection safety：[https://developers.openai.com/api/docs/guides/agent-builder-safety](https://developers.openai.com/api/docs/guides/agent-builder-safety)
- 通用 safety / HITL：[https://developers.openai.com/api/docs/guides/safety-best-practices](https://developers.openai.com/api/docs/guides/safety-best-practices)
- MCP/connector approvals：[https://developers.openai.com/api/docs/guides/tools-connectors-mcp](https://developers.openai.com/api/docs/guides/tools-connectors-mcp)

### Anthropic

- Prompt engineering 前置条件：[https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/overview](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/overview)
- 最新模型 prompting/migration 实践：[https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices)
- Tool use：[https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview](https://platform.claude.com/docs/en/agents-and-tools/tool-use/overview)
- Tool definitions：[https://platform.claude.com/docs/en/agents-and-tools/tool-use/define-tools](https://platform.claude.com/docs/en/agents-and-tools/tool-use/define-tools)
- Prompt caching：[https://platform.claude.com/docs/en/build-with-claude/prompt-caching](https://platform.claude.com/docs/en/build-with-claude/prompt-caching)
- Eval success criteria：[https://platform.claude.com/docs/en/test-and-evaluate/develop-tests](https://platform.claude.com/docs/en/test-and-evaluate/develop-tests)
- Prompt injection 防御：[https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/mitigate-jailbreaks](https://platform.claude.com/docs/en/test-and-evaluate/strengthen-guardrails/mitigate-jailbreaks)

### Model Context Protocol

- 最新稳定工具规范（2025-11-25）：[https://modelcontextprotocol.io/specification/2025-11-25/server/tools](https://modelcontextprotocol.io/specification/2025-11-25/server/tools)
- 最新稳定授权规范（2025-11-25）：[https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization](https://modelcontextprotocol.io/specification/2025-11-25/basic/authorization)
- MCP Security Best Practices：[https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices](https://modelcontextprotocol.io/docs/tutorials/security/security_best_practices)
- MCP Authorization 教程：[https://modelcontextprotocol.io/docs/tutorials/security/authorization](https://modelcontextprotocol.io/docs/tutorials/security/authorization)
