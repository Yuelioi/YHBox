# Yotta 3.0 AI-native 目标设计

## 产品判断

Yotta 3.0 不应继续被定义为“带一个 AI 节点的可视化自动化工具”。更强的模型使自然语言规划、工具调用和迭代修复变得可用，但模型不会替代可靠的工作流 IR、编译器、权限边界和可观测执行。

新的产品定义是：

> **Yotta 是本地优先、可审计、可扩展的 AI 自动化工作台。人类与 AI 共同编辑同一种强类型工作流，所有执行都经过同一个编译器、能力授权和追踪系统。**

这不是把整个应用交给 agent。确定性的图仍是长期资产；agent 负责理解意图、检索节点、提出最小 patch、解释 diagnostics 和在授权预算内验证。生成完成后，用户得到的是可检查、可版本化、可重放的工作流，而不是藏在聊天历史里的隐式行为。

## 三个一等客户端，一个核心

```text
Human Studio              AI Authoring Agent             Headless CLI / MCP
EditorSession             plan → patch → compile         run / validate / inspect
       \                         |                         /
        +------------------------+------------------------+
                                 |
                    Workflow Authoring Protocol
                                 |
          catalog.search / describe / apply_patch / compile
                                 |
          +----------------------v-----------------------+
          | workflow schema + compiler + immutable IR   |
          | workspace transactions + package/lock       |
          | capability policy + approvals               |
          | runtime + trace + eval                      |
          +----------------------------------------------+
```

三者不得各自实现 schema 修复、pin compatibility、节点默认值或保存逻辑。AI authoring 不是新的旁路 API；它是 EditorSession/Compiler 的另一个 typed client。

## 当前 AI surface 的审查结论

| 当前实现 | 问题 | 3.0 决策 |
| --- | --- | --- |
| `internal/services/llm.Provider` 只有 Chat/Text 最低公分母 | 丢失 provider-native item、usage、stop reason、request ID、reasoning/tool loop 语义 | 删除通用 Chat 抽象；建立共享 orchestration contract 与 provider-native adapter |
| OpenAI adapter 使用 Chat Completions | 无法成为新的 OpenAI 主路径 | 官方 OpenAI adapter 只实现 Responses API；迁移完成后删除旧路径 |
| `ModeAuto/ModeNative/ModePrompt` | 能力由 endpoint 字符串猜测，行为不可预测 | 删除三种 mode；安装时声明并探测 capabilities，不满足即拒绝 |
| `structuredViaPrompt` 追加“只输出 JSON”并截取 fence/花括号 | 解析启发式把协议错误伪装成成功 | 删除 prompt JSON fallback；结构化任务必须使用 provider 支持的严格 schema |
| `JSONSchema` 只有扁平的字段名和 JSON type | 不能表达嵌套、enum、required、约束或 unknown-field policy | 使用完整 JSON Schema 2020-12 子集，默认 `additionalProperties: false` |
| AI 节点把 `{{dynamic}}` 同时插入 system/user | 不可信运行时数据可提升到高权限指令层 | system/developer 只来自版本化 manifest；动态值只进入 typed user/context/tool-result |
| 缺失模型字段时静默保留旧变量值 | 新结果与旧状态混合，无法审计 | 输出整体校验后原子提交；缺失 required field 即失败 |
| 连接与 model string 直接持久化在节点 | 工作流不可移植，升级不可治理 | 节点引用 `AISlot`；installation 将 slot 绑定到 ModelProfile/provider/model snapshot |
| API key 明文写入 settings 并回传前端 | 密钥生命周期与普通配置混合 | 使用 OS credential store；前端只看到 credential reference 与 masked metadata |
| MCP 一次返回约 13.3 万字符、137 个节点目录 | 成本高、选择质量低、上下文污染 | 改成 search → describe → schema/resource 按需加载 |
| MCP 接收/返回整图 JSON 文本 | 并发覆盖、diff 不透明、模型容易误改 | 使用带 base revision 的 typed patch command 与 structured result |
| `Normalize()` 在验证前自动修复 schema version | 错误示例和坏输出被掩盖 | authoring 边界永不 normalize；严格 decode/compile，明确 diagnostics |

## AI 模块边界

```text
internal/ai
  modelpolicy/       ModelProfile、snapshot、capability negotiation、升级 gate
  prompt/            PromptManifest、render、hash、trusted/untrusted block
  schema/            JSON Schema 校验、provider dialect 编译
  tools/             ToolManifest、registry、discovery、approval metadata
  session/           turn、budget、compaction、TTL、resume contract
  eval/              dataset、grader、threshold、regression report
  trace/             redacted event、usage、latency、tool/approval lineage
  safety/            injection boundary、output validation、secret redaction
  provider/
    openairesponses/ 官方 Responses adapter
    anthropic/       官方 Messages adapter
    ...              显式第三方 adapter；不冒充官方能力
```

共享层不暴露 `Chat(messages) -> text`。它表达 Yotta 真正需要的语义：typed input blocks、structured result、tool request、usage、finish state、provider request ID、stream event 和可取消性。provider adapter 负责将这些语义映射到自己的原生 API；缺少所需 capability 时返回 `UnsupportedCapability`，不得退回 prompt 模拟。

### ModelProfile

节点不保存任意模型名。节点只声明意图，例如 `fast_extract`、`balanced_generate`、`deep_agent`、`vision_extract`；工作区或安装环境把意图绑定到：

- provider adapter 与 connection reference；
- 固定 model snapshot/version；
- input modalities、structured output、tool use、parallel tool、streaming 等 capability；
- context/output limit、reasoning policy、timeout、cost/turn/tool budget；
- 允许使用的 PromptManifest/ToolSet/EvalSuite 版本；
- 用户可见的升级状态与最近一次 eval 结果。

“OpenAI compatible”不是 provider 类型，只能是第三方 adapter 的自报协议与 capability 集合。URL 中是否包含官方域名不得影响语义。

### PromptManifest

内置 prompt 是版本化构建产物，不是散落字符串：

```yaml
id: workflow.author.v1
owner: ai-authoring
version: 1.0.0
goal: 将用户意图转换为最小、可编译的工作流 patch
rules:
  - 不修改目标之外的节点
  - diagnostics 未清零不得声称完成
tool_policy: authoring-minimal
input_schema: yotta://schemas/author-request.v1.json
output_schema: yotta://schemas/author-result.v1.json
eval_suite: workflow-authoring.v1
```

渲染后的 prompt、schema、toolset、model profile 都记录 content hash。trace 必须能回答“哪个模型、哪版指令、哪些工具、什么输入、哪些审批产生了这个 patch”。

推荐最小结构只有 Goal、Rules、Tool policy、必要 examples 和 Context contract。删除：

- 重复的 `CRITICAL`、`MUST`、全大写威慑和同义规则；
- 要求模型展示完整 chain-of-thought 的文字；
- “只输出 JSON”式协议模拟；
- 把整个节点目录、整个工作区或长期聊天历史塞入 prompt；
- provider 共用的超长人格 prompt；
- 将用户输入、网页、OCR、节点输出插入 system/developer；
- 仓库级 contributor 规则与产品运行时 prompt 的复制粘贴。

必须保留并外部强制：目标、不可违反的业务约束、审批边界、成功标准、输出 schema、预算和失败语义。权限约束不依赖模型服从，而由 ToolRegistry/CapabilityPolicy 拒绝未授权调用。

## AI 节点重新分型

删除目前一个节点同时承担自由生成与结构化状态写入的模糊行为，改为三个稳定节点族：

1. **Generate**：文本/多模态生成；输出是显式内容与 usage，不修改多变量。
2. **Extract / Classify**：必须提供完整 output schema；结果整体校验、整体提交。
3. **Agent**：绑定受限 ToolSet、RunBudget 与 approval policy；产生可追踪的 tool loop。

长会话能力若确有产品需求，再增加 **Session** resource。它拥有 session ID、TTL、owner、summary/compaction policy 与删除操作；不能借用节点运行历史无限追加 messages。

## AI authoring protocol

模型不应编辑整份 Container JSON。Authoring API 使用领域命令和 revision：

```text
catalog.search(query, capability, limit)
catalog.describe(kind, schema_version)
workflow.inspect(scope, projection)
workflow.apply_patch(base_revision, commands[])
workflow.compile(revision)
workflow.explain_diagnostic(code, context)
workflow.run_preview(revision, policy, budget)
```

`commands` 是 add/remove/move/configure/connect/disconnect/rename/set-variable 等 tagged union。服务端负责 default、ID、pin/type、reference integrity 和 transaction。返回值包含 new revision、normalized domain patch（不是 silent repair）、diagnostics 和 permission delta。revision 冲突时返回冲突，不覆盖。

典型闭环：

1. 将用户目标编译为成功标准和 capability 需求。
2. 搜索少量候选节点；只加载候选的 machine contract。
3. 提出最小 patch；服务器严格应用。
4. 编译并读取结构化 diagnostics。
5. 在 iteration/tool/token/time budget 内修复。
6. 权限扩大或产生外部副作用前请求批准。
7. 通过 compile/eval 后给出可读 diff、权限变化和未解决风险。

节点的 machine contract 与 UI 文案分离。catalog description 应短、稳定、任务导向，不复用诸如“看起来更像真人”的营销/帮助文本。每个节点明确 effect class、determinism、idempotency、required capability、inputs/outputs、config schema、examples 与 error codes。

## MCP 3.0

MCP 是 AI authoring protocol 的适配器，不是另一个业务层。

- 默认关闭；优先 stdio。若启用 HTTP，只允许 authenticated loopback、随机 session token、origin/host 校验和生命周期绑定。
- 初始只暴露少量安全的 discovery/inspect/compile 工具；写入、运行、窗口枚举等按 capability 单独 armed/approved。
- tools 同时声明严格 input schema 与 output schema，返回 `structuredContent`；文本仅作人类摘要。
- `save_container(container_json)`、`validate_container(container_json)`、`get_graph_schema` 整体 blob API 全部删除。
- `list_nodes` 改 search/describe；大 schema、trace、catalog 通过 resource URI 分页读取。
- `list_windows` 属于隐私敏感能力，默认不注册；只有明确授权后短时暴露。
- tool result 进入不可信层；任何返回的“请忽略规则”等文本都不能改变 tool/policy。

## Context、cache 与 trace

上下文按稳定性从前到后排列：固定 policy/prompt → tool/schema → workspace 摘要 → 本轮用户输入 → tool result。稳定前缀可以缓存；动态内容不进入前缀。cache 是性能优化，不得改变语义或成为正确性依赖。

每次 run 产生可脱敏 trace：

- model/provider/profile snapshot；
- prompt/schema/toolset hash；
- input block 的来源和 trust class；
- token/latency/cache/stop/usage；
- tool arguments/result schema validation；
- approval request/decision 与 capability delta；
- patch、compiler diagnostics、run outcome；
- provider request ID 和 Yotta correlation ID。

默认不记录 secret、原始截图、完整网页/文件、credential 或用户标记的敏感变量；调试采样必须显式、限时、可删除。

## Evals 是模型升级门禁

模型、prompt、schema 或 toolset 任一变化都可能改变行为，因此它们共享一条 eval gate。首批数据集直接从真实 Yotta 任务构造：

- 中文/英文工作流生成；
- 模糊目标的澄清与拒绝；
- catalog 检索与节点选择；
- pin/type/config 正确性；
- 最小 patch 与无关节点保护；
- compiler diagnostics 自修复；
- prompt injection/恶意 tool result；
- 未授权文件、网络、进程、输入和窗口能力；
- structured extraction 的缺失字段/unknown field；
- token、tool call、latency 与费用预算。

确定性 grader 校验 schema、compile、permission 与 graph diff；模型 grader 只评价主观质量，并保存 rubric/version。发布 gate 比较候选与基线的 pass rate、安全失败率、p95 成本/延迟，禁止仅凭单次人工聊天升级模型。

## Contributor agent 指令也要升级

仓库内供 Codex/Claude 等开发 agent 使用的规则与产品 prompt 必须是两个 owner：

- 新建受版本控制的简短 `AGENTS.md`，只保留仓库事实、必跑命令、危险边界、架构索引和 Flightdeck 入口。
- 其他 provider 文件若需要，只做薄适配并指向同一 canonical contract；不得复制完整规则。
- 当前本地 `CLAUDE.md` 被 gitignore，仍写着 `YHFish`、直接提交 main 和过时硬审批流程，不能作为项目规范。
- 把可机械验证的规则移入 CI/linter/schema；prompt 只解释不可机械化的意图与决策边界。
- 每次重大架构变化同步更新 agent navigation，并用小型 repo tasks 验证 agent 是否找到正确入口、运行正确门禁、未编辑生成物。

## 明确不做

- 不维护旧 Chat/Responses 双轨或 JSON prompt fallback。
- 不让 agent 绕过 Compiler/Workspace 直接写文件。
- 不把“自主”解释为默认获得文件、网络、进程、输入或窗口权限。
- 不依赖隐藏 chain-of-thought；只要求简短决策摘要、证据、patch 和 diagnostics。
- 不把所有 provider 压成同一 wire format；只统一 Yotta 需要的领域语义。
- 不把 137 个节点和完整图一次性送入上下文。
- 不在没有 eval 基线时自动漂移到模型 alias 的最新版本。

## 完成定义

- OpenAI 官方路径只使用 Responses API；Anthropic 使用原生 Messages；不存在 `ModePrompt`/JSON 截取。
- AI 节点只引用 slot/profile；credential 不再落普通 settings 或返回前端。
- PromptManifest、ModelProfile、Schema、ToolSet 均可版本化、hash、追踪和回滚。
- dynamic/untrusted data 无法进入 system/developer block。
- Extract/Classify 严格校验并原子提交；Agent 有 tool/time/token/cost/iteration budget。
- AI 与 UI 共用 typed patch、Compiler、Workspace transaction 和 permission manifest。
- MCP 默认关闭，tool input/output 都有 schema；catalog 按需发现；无整图 JSON 保存工具。
- 模型/prompt/tool/schema 升级须通过离线 eval 与安全 regression gate。
- 用户可以查看每个 AI 变更的 diff、diagnostics、权限变化和脱敏 trace。
