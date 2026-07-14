# 主流可视化节点与工作流系统设计实践

> 状态：非规范研究资产。它记录主流系统证据与候选做法；Yotta 已接受的最终规则以对应 Wayfinder ticket 的 Resolution 和 ADR/schema 为准，尤其是 3.1 已决定删除 wildcard `any` 且不迁移未发布的旧 workflow。

> 研究日期：2026-07-14
> 范围：节点纯度、执行/数据端口、类型与 schema、插件与跨平台 capability、版本兼容、确定性执行、错误模型、数据血缘，以及 catalog、编辑器和文档生成。
> 资料原则：只采用官方文档、官方源码仓库或正式规范；每个关键结论在结论附近给出来源。

## 摘要

对 Yotta 最有价值的主流共识不是某一种画布样式，而是以下契约边界：

1. **执行边和数据边是不同语义，不得互相推断。**纯数据节点没有执行端口；端口是否存在必须由 canonical node contract 明确声明，编辑器不得补一个猜测的 `out`。
2. **“无副作用”和“确定性”是两个属性。**纯节点必须无副作用，但读取时间、随机数或不稳定环境仍可能不确定；二者应分别建模。
3. **节点描述应成为单一事实来源。**运行时 registry、画布端口、配置表单、连接校验、catalog、文档、示例与弃用提示都应从同一份版本化 contract 生成。
4. **破坏性升级必须显式版本化。**允许重构当前实现，不等于允许默默改变已保存工作流的含义；节点实例必须绑定契约版本，并通过旧实现或逐步迁移恢复。
5. **平台和权限是调度约束，不是运行时碰运气。**OS、架构、宿主应用、网络、文件、进程、GUI 自动化与凭据能力应在执行前匹配并授权。
6. **确定性编排与外部副作用应分层。**图状态转换可重放；AI、网络、文件、进程和宿主自动化通过 effect adapter 执行，并记录输入、结果、失败与重试身份。
7. **业务数据、错误和运行状态应分通道。**错误需要结构化、可定位、可判断是否可重试；状态更新不能伪装成普通输出值。
8. **输出应保留来源。**数据血缘是局部重跑、错误定位、表达式解析和可解释调试的基础。
9. **自动生成文档是可行的，但前提是 schema 包含语义注解。**只有类型名不足以生成可用文档；还需要说明、示例、默认值语义、弃用信息、能力和错误契约。

## 系统对照

| 系统/规范 | 主要借鉴点 | 不宜直接照搬的部分 |
| --- | --- | --- |
| Unreal Blueprint | Pure/Impure、Exec/Data pin 的清晰分离、反射驱动节点表面 | 面向游戏对象和帧循环的具体语义 |
| Blockly | 值块/语句块分离、连接检查、JSON 定义、可插拔序列化 | 主要目标是代码生成，不是耐久工作流执行 |
| Node-RED | 插件包装、运行状态、可捕获错误、节点内帮助规范 | 单一消息总线和最多一个输入口不适合照搬为通用 typed graph |
| n8n | 节点 description、存量节点版本、结构化错误、item lineage | item-list 数据模型不一定适合所有桌面自动化节点 |
| Temporal | 可重放编排、effect/activity 隔离、版本演进 | 分布式耐久执行的全部复杂度未必都需要引入 |
| GitHub Actions | runner capability 匹配、权限最小化、插件版本固定 | YAML step 模型不是通用可视化数据流类型系统 |
| JSON Schema / OpenAPI | 结构验证、注解、复用，以及 UI/文档/工具生成 | 它们不定义执行端口、纯度、重试和 capability，需要扩展词汇 |

## 1. 纯节点、非纯节点与端口语义

### 1.1 纯节点不应有执行端口

Unreal 将 Pure Function 定义为不会直接影响 world 或对象的操作；其结果被使用时才求值。相对地，可变/非纯函数具有执行 pin。Epic 的 RigVM API 也直接把 pure 描述为“没有 execute pins”，mutable 描述为“有 execute pins”。Pure Function 还可能针对每个连接的消费者分别调用，而不是天然只计算一次。[Epic：Function Calls—Pure](https://dev.epicgames.com/documentation/en-us/unreal-engine/function-calls-in-unreal-engine)；[Epic：Functions](https://dev.epicgames.com/documentation/en-us/unreal-engine/functions-in-unreal-engine?lang=en-US)；[Epic：Set Function Is Pure](https://dev.epicgames.com/documentation/en-us/unreal-engine/BlueprintAPI/RigVMController/SetFunctionIsPure?lang=en-US)

Blockly 用另一套术语表达同一条边界：Value Block 有 output connection，像表达式一样产生值；Statement Block 没有值输出，像语句一样生成控制行为。其生成器明确区分 `valueToCode` 和 `statementToCode`，next connection 也不属于 value input。[Blockly：Block-code generators](https://developers.google.com/blockly/guides/create-custom-blocks/code-generation/block-code)

**对 Yotta 的约束：**

- `pure-data` 节点只能声明 data inputs/outputs，执行端口集合必须是空数组。
- `effect` 或 `control` 节点才允许声明 exec inputs/outputs。
- 后端 contract 中的空端口集合是有效且有意义的值，前端不得把空集合解释为缺省 `['out']`。
- 画布连接类型、验证和运行时求值路径都必须区分 `exec` 与 `data`；“同名 out”不能承担两套语义。
- 纯节点连接多个消费者时，运行时必须明确采用“每次拉取重算”还是“单次执行内 memoize”，并用测试锁定；不能依赖 UI 观察推断。

### 1.2 纯度不等于确定性

Unreal 的 pure 重点是“不直接修改状态”；Temporal 对 Workflow 的要求更强：工作流代码必须确定且无副作用，外部交互放入 Activity。Temporal 的架构以事件历史重放恢复状态，并要求 Activity 要么幂等、要么明确不可重试。[Temporal：Architecture](https://github.com/temporalio/temporal/blob/main/docs/architecture/README.md)

因此节点至少需要两个独立维度：

| 维度 | 示例值 | 含义 |
| --- | --- | --- |
| `effects` | `none` / `filesystem` / `network` / `process` / `host-automation` | 是否及如何改变外部世界 |
| `determinism` | `deterministic` / `recorded` / `nondeterministic` | 相同输入能否稳定得到相同输出，或是否依靠记录回放 |

例如“读取当前时间”可以无写入副作用，但不是确定性函数；“写文件”有副作用，即使写入内容完全由输入确定，也不是纯节点。只有 `effects: none` 且 `determinism: deterministic` 的节点才适合无条件缓存、重算和并行化。

### 1.3 节点表面必须由契约投影

n8n 的节点类包含 `description` 对象来定义节点，程序式节点另有 `execute()`；声明式节点则通过描述内的 routing 表达行为。[n8n：Structure of the node base file](https://docs.n8n.io/integrations/creating-nodes/build/reference/node-base-files/structure/)

GitHub Actions 同样要求 action metadata 明确声明名称、说明、inputs、outputs 与运行方式；输出是供后续 action 作为输入消费的具名参数。[GitHub Actions：Metadata syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax)

**审查门禁：**同一节点的后端端口、前端端口、表单字段、默认值、catalog 条目和文档之间不应存在手写平行定义。若前端需要 presentation-only 信息，也应以稳定扩展字段挂在 canonical contract 上，而不是复制运行语义。

## 2. 类型系统与 schema

### 2.1 连接检查要在建边、加载和执行前一致

Blockly 的连接检查分为 safety、type 和 drag 三层；safety 防止跨 workspace、同一 block 自连以及两个 next connection 等坏状态，type 阻止例如 string 输出接到 number 输入。官方建议通常不要覆盖 safety checks。[Blockly：Custom connection checkers](https://developers.google.com/blockly/guides/create-custom-blocks/inputs/connection_checker)

Blockly 对类型还有一条实用约定：输入列出其接受的所有类型，输出尽量只声明实际返回的精确类型；接受任意类型必须显式使用 `null`，而不是依赖隐式 fallback。[Blockly：Connection check playbook](https://developers.google.com/blockly/guides/create-custom-blocks/inputs/connection-check-playbook)

**对 Yotta 的约束：**

- 端口类型至少支持精确类型、union、optional/nullability、collection 和显式 `any`。
- “可连接”判定必须由共享库实现，并同时用于拖线、反序列化校验、运行前校验和 CLI/文档示例校验。
- block/node type identifier 与 data type identifier 是不同命名空间，不能因为名字相同就连接。
- 自动转换必须是显式 conversion node 或可见的编译产物，不能静默改变值。
- `any` 是有意放宽的 contract，而不是未知类型的兜底。

### 2.2 JSON Schema 适合描述数据，不足以单独描述节点执行

JSON Schema Draft 2020-12 将结构验证 assertion 与 `title`、`description`、`default`、`examples`、`deprecated`、`readOnly`、`writeOnly` 等 annotation 放在同一体系；这些 annotation 面向文档和 UI。规范同时明确，`format` 在默认 vocabulary 中是 annotation，不保证执行断言式验证。[JSON Schema 2020-12 Validation](https://json-schema.org/draft/2020-12/json-schema-validation)

另一个容易踩坑的事实是：`default` 不会由验证器自动填充缺失值，它只是给非验证工具的提示。[JSON Schema：Annotations](https://json-schema.org/understanding-json-schema/reference/annotations)

**建议：**每个 data pin 和配置字段使用 JSON Schema 子集或兼容 dialect；Yotta 自己扩展节点级词汇来描述：

- pin kind 与方向；
- effect、determinism、retry 与 idempotency；
- capability 和权限；
- secret/credential binding；
- UI widget hint；
- 版本、迁移和弃用；
- 错误类型与错误出口；
- 文档、示例和平台限制。

不要把 `format: path`、`format: command` 等自定义格式误认为安全验证。JSON Schema 规范指出，自定义 format 的跨实现支持不能被假定；涉及路径、命令、URL 和凭据的语义验证必须由 Yotta 明确实现。[JSON Schema 2020-12 Validation：Custom formats](https://json-schema.org/draft/2020-12/json-schema-validation)

### 2.3 schema 必须有版本和 dialect

JSON Schema 通过 `$schema` 标明 dialect，2020-12 还把功能拆成 vocabulary；OpenAPI 也发布明确版本与 schema iteration，并提醒 schema 无法捕获全部规范违规，冲突时规范文本优先。[JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12)；[OpenAPI Specification versions](https://spec.openapis.org/oas/)

**对 Yotta 的约束：**节点 contract 必须携带 `contractVersion` 和 schema dialect；解析器不能根据字段“看起来像新版”来猜版本。自定义词汇要有命名空间和兼容策略。

## 3. 插件、跨平台 capability 与权限

### 3.1 平台支持应是可调度条件

GitHub Actions 的 `runs-on` 使用 runner labels 和 groups 选择执行环境；标签数组要求 runner 同时满足全部标签，例如 `self-hosted + linux + x64 + gpu`。runner group 还是访问边界，而不只是展示分类。[GitHub Actions：Choosing the runner](https://docs.github.com/en/actions/how-tos/write-workflows/choose-where-workflows-run/choose-the-runner-for-a-job)；[GitHub Actions：Runner groups](https://docs.github.com/en/actions/concepts/runners/runner-groups)

Yotta 可采用同样的匹配思路，但 capability 应比 OS 更细：

```yaml
requires:
  os: [windows]
  arch: [amd64, arm64]
  hostApps:
    - id: after-effects
      version: ">=24 <27"
  capabilities:
    - process.spawn
    - filesystem.read
    - host.after-effects.script
  permissions:
    network: none
    filesystem:
      read: [workspace, selected-assets]
      write: [workspace-output]
```

调度前应产生三种明确结果：`eligible`、`unsupported`、`permission-required`。缺少能力时不应把节点照常放入执行队列，等 adapter 抛出模糊错误。

### 3.2 capability 声明不等于授权

GitHub 建议通过 workflow/job 的 `permissions` 给 `GITHUB_TOKEN` 最小所需访问；runner groups 还能限制哪些仓库可以使用某组 runner。[GitHub Actions：Automatic token authentication](https://docs.github.com/en/actions/security-for-github-actions/security-guides/automatic-token-authentication)；[GitHub Actions：Runner groups](https://docs.github.com/en/actions/concepts/runners/runner-groups)

**对 Yotta 的约束：**

- 节点 contract 声明“需要什么”，workflow/user policy 决定“授予什么”。
- credential 以引用注入，不能成为普通可串联明文 data pin。
- 文件、网络、进程、MCP/package 与桌面自动化输入均视为不可信。
- capability 检查必须发生在 adapter 调用前，并将授权决定写入 run record。
- 预览/文档模式不能因为渲染节点就触发 capability 或副作用。

### 3.3 插件要同时约束宿主版本和实现版本

Node-RED 节点包通过 `package.json` 的 `node-red.nodes` 声明运行时入口，也可以用 `node-red.version` 声明所需宿主版本；README 应列出能力和前置条件。[Node-RED：Packaging](https://nodered.org/docs/creating-nodes/packaging)

GitHub Actions 建议固定 action 版本；对稳定性和安全性而言，完整 commit SHA 最安全，固定 major version 是兼容性与补丁更新之间的折中，跟随默认分支可能被破坏性更新影响。[GitHub Actions：Metadata syntax—runs.steps.uses](https://docs.github.com/en/actions/reference/workflows-and-actions/metadata-syntax)

Yotta 至少应区分：

| 标识 | 用途 |
| --- | --- |
| `nodeType` | 稳定的逻辑节点身份，例如 `text.concat` |
| `contractVersion` | 已保存图绑定的端口、配置和语义版本 |
| `implementationVersion` | 具体代码/包构建版本，可含兼容修复 |
| `hostApiRange` | 插件兼容的 Yotta host API 范围 |
| `artifactDigest` | 可选的不可变包摘要/签名身份 |

## 4. 版本兼容与破坏性升级

n8n 的规则非常直接：工作流以 v1 创建并保存后继续加载 v1；发布 v2 后，新建工作流才默认使用最新版。它同时提供 light versioning 和独立版本实现，后者的基类只保存默认版本和版本映射，功能留在各版本实现中。[n8n：Node versioning](https://docs.n8n.io/integrations/creating-nodes/build/reference/node-versioning/)

Blockly 推荐新项目使用 JSON 序列化；其反序列化有明确顺序，并允许插件注册带 priority 的 serializer。类型、属性、extra state、父连接、字段、输入块和 next block 不是随意恢复的。[Blockly：Save and load](https://developers.google.com/blockly/guides/configure/web/serialization)

Temporal 的事件溯源模型进一步说明，存量执行依赖旧事件历史被新代码正确重放；不兼容变化要通过 patch/versioning 或保留旧 worker 处理。其官方架构要求工作流可由追加式历史恢复。[Temporal：Architecture](https://github.com/temporalio/temporal/blob/main/docs/architecture/README.md)

**Yotta 的推荐加载算法：**

1. 读取 graph schema version。
2. 对每个节点读取精确的 `nodeType + contractVersion`。
3. 若对应 contract/实现仍存在，按原版本加载。
4. 若不存在，只允许执行已登记、可测试的逐版本 migration；不得直接解释为 latest。
5. migration 先在内存副本执行，完成全图验证后再保存。
6. 保存时记录迁移 receipt，包括原版本、目标版本、警告和不可逆变化。
7. 端口被删除或改型时，旧 edge 必须产生明确诊断；不能静默断线。
8. catalog 对新节点默认展示 latest，但已保存实例继续显示自身版本及弃用状态。

允许破坏性更新时，应优先破坏**代码内部接口**以收敛架构，同时保护**已持久化工作流语义**。若产品决定不兼容旧图，也应提供一次性迁移工具和可读诊断，而不是隐式失败。

## 5. 确定性执行、重试与数据血缘

### 5.1 编排器记录决定，adapter 执行副作用

Temporal 将 Workflow 与 Activity 分开：Workflow 负责可重放的确定性编排，Activity 承担外部副作用；Activity 要么能安全重试，要么标为不可重试。[Temporal：Architecture](https://github.com/temporalio/temporal/blob/main/docs/architecture/README.md)

对应到 Yotta：

- scheduler 只推进图状态、分支选择、依赖满足和取消传播；
- AI、网络、文件、进程、MCP、UE/AE 等宿主自动化全部通过 effect adapter；
- 每次 effect 调用写入 `operationId`、规范化输入摘要、capability/授权、开始与结束时间、结果摘要、错误和 retry attempt；
- 可重试副作用需要 idempotency key 或等效防重机制；
- 纯节点结果缓存的 key 至少包含 node contract version、实现兼容标识、规范化输入和相关静态配置；
- 非确定性但可记录的值（时间、随机数、AI 响应）在 replay 时使用记录值，而不是重新调用。

### 5.2 输出必须带来源关系

n8n 的 item linking 要求节点输出记录其来源输入；缺少 `pairedItem` 会令下游表达式无法可靠找到对应上游项，多输入节点还要记录 input index。[n8n：Item linking for node creators](https://docs.n8n.io/data/data-mapping/data-item-linking/item-linking-node-building/)

Yotta 的 run value 建议包含：

```json
{
  "value": "...",
  "type": "string",
  "producedBy": {
    "runId": "...",
    "nodeId": "...",
    "portId": "result",
    "attempt": 1
  },
  "derivedFrom": [
    { "nodeId": "...", "portId": "text", "item": 0 }
  ]
}
```

血缘不必永久复制完整值，但稳定引用应能支持：查看某输出来自哪些输入、定位某 item 的错误、判断局部重跑失效范围、解释 AI/脚本最终参数如何组装。

## 6. 错误模型、状态与控制策略

### 6.1 错误类别应可操作

n8n 区分 `NodeApiError` 和 `NodeOperationError`：前者处理 HTTP、认证、限流和外部服务失败；后者处理验证、配置、缺参、数据转换和工作流逻辑错误，并可带 item index。[n8n：Error handling](https://docs.n8n.io/integrations/creating-nodes/build/reference/error-handling/)

Node-RED 进一步区分只进日志的错误、可触发 Catch flow 的错误，以及令 runtime 停止的 uncaught exception；可捕获错误包含来源节点信息。连接状态等非错误情况则通过独立 status 事件表达。[Node-RED：Handling errors](https://nodered.org/docs/user-guide/handling-errors)；[Node-RED：Node status](https://nodered.org/docs/creating-nodes/status)

**Yotta 统一错误最小字段：**

```yaml
category: validation | configuration | capability | permission | external | timeout | cancelled | internal
code: stable.machine.readable.code
message: 面向用户的简短说明
detail: 技术原因，默认可折叠
source:
  runId: ...
  nodeId: ...
  portId: ...
  item: 0
attempt: 1
retryable: true
retryAfter: 5s
cause: ...
remediation: ...
```

### 6.2 数据、错误、状态和控制出口不要混为一个 `out`

推荐分成四种通道：

| 通道 | 例子 | 是否成为普通 data pin |
| --- | --- | --- |
| data | 拼接结果、文件路径、模型响应 | 是 |
| exec/control | 成功后继续、条件分支 | 否，使用 exec edge |
| error | 认证失败、缺参、超时 | 否，进入错误策略或显式 error edge |
| status/progress | 连接中、50%、等待宿主 | 否，作为 run event |

运行策略至少要明确定义 `fail-fast`、`continue-on-error`、`retry`、`fallback/error-edge`、`cancel` 与超时传播。Node-RED 的经验表明，未捕获异步错误会令整个 flow 状态不可知，因此 adapter 边界必须捕获并规范化所有错误。[Node-RED：Creating nodes](https://nodered.org/docs/creating-nodes/)

## 7. 从 contract 自动生成 catalog、编辑器与文档

JSON Schema 的 metadata vocabulary 明确把 `title` 和 `description` 用于 UI 装饰，并提供 default、examples、deprecated、readOnly、writeOnly 等注解。[JSON Schema 2020-12 Validation：Metadata](https://json-schema.org/draft/2020-12/json-schema-validation)

OpenAPI 表明，机器可读描述可以承载 summary、description、examples、参数、请求/响应 schema 和安全要求；同一描述被广泛用于客户端生成、文档生成、服务端路由和 API 测试。[OpenAPI 3.1.2 Specification](https://spec.openapis.org/oas/v3.1.2.html)；[OpenAPI：Getting started](https://learn.openapis.org/)

Node-RED 对节点帮助也规定了稳定结构：简介、Inputs、Outputs、Details，多输出逐项描述，并注明消息属性类型。[Node-RED：Node help style guide](https://nodered.org/docs/creating-nodes/help-style-guide)

**Yotta 可生成的产物：**

1. **运行时 registry：**节点 ID、版本、构造器/evaluator、adapter 绑定。
2. **前端 catalog：**名称、分类、搜索词、平台可用性、弃用与实验标记。
3. **画布节点：**严格按 contract 渲染 exec/data pins，不补默认 pin。
4. **配置表单：**由配置 schema、enum、default 和 widget hint 生成。
5. **静态验证器：**边兼容、必填配置、能力、权限、图结构和版本检查。
6. **节点参考文档：**用途、输入、输出、错误、能力、平台、示例、版本历史。
7. **机器可读 catalog：**供 AI、CLI、MCP 或第三方编辑器发现节点。
8. **契约测试 fixture：**从 examples 生成解析、验证和文档 smoke tests。

生成器必须拒绝以下不完整 contract，而不是生成误导性页面：

- 端口缺少稳定 ID 或类型；
- effect 节点没有 capability；
- 配置字段缺少用户说明；
- `default` 不符合自身 schema；
- 标为 pure 但声明外部 capability；
- 标为 deterministic 却读取未记录的时间、随机或环境；
- 删除/改型端口却没有 contract version 变化；
- 错误只有自由文本而没有稳定 code/category。

## 8. 推荐的 canonical node contract 轮廓

下面是综合上述系统后给 Yotta 的建议轮廓，不是某个外部标准的原样复制：

```yaml
schema: yotta.node/v1
nodeType: text.concat
contractVersion: 2
implementationVersion: 3.1.0+build.42

display:
  title: 拼接
  category: text
  summary: 按顺序拼接两个字符串
  description: ...
  tags: [string, concat]

semantics:
  kind: pure-data
  effects: none
  determinism: deterministic
  evaluation: pull
  cache: per-run

ports:
  execInputs: []
  execOutputs: []
  dataInputs:
    - id: a
      title: A
      schema: { type: string }
      required: true
    - id: b
      title: B
      schema: { type: string }
      required: true
  dataOutputs:
    - id: result
      title: 结果
      schema: { type: string }

configuration:
  schema:
    type: object
    additionalProperties: false

runtime:
  hostApi: ">=3.1 <4"
  platforms: [windows, linux, darwin]
  capabilities: []
  retry: never

errors:
  - code: text.concat.invalid-input
    category: validation

docs:
  examples:
    - name: 拼接脚本片段
      inputs: { a: "foo", b: "bar" }
      output: "foobar"

lifecycle:
  status: stable
  deprecated: false
  migrations:
    from: [1]
```

关键规则是：不存在的端口使用空数组表示；`null`、缺字段与空数组的含义必须在 meta-schema 中区分。前端只能投影 contract，不能通过节点类别、名字或历史惯例补端口。

## 9. 深度审查建议清单

### P0：会改变执行正确性或安全边界

- 搜索所有前端/adapter 的默认 exec pin 注入和端口猜测。
- 检查 pure 节点是否可能调用文件、网络、进程、时间、随机、环境或宿主自动化。
- 检查 graph 加载是否把未知旧版本静默映射到 latest。
- 检查 capability/权限是否在执行前统一验证。
- 检查 async error 是否都被捕获并转成结构化错误。
- 检查取消、超时和重试是否可能重复不可幂等副作用。

### P1：会阻止可扩展和可靠演进

- 盘点后端、前端、文档是否存在平行节点定义。
- 为 node、graph、plugin/implementation 建立分离的版本身份。
- 统一建边、加载、运行前的类型兼容算法。
- 明确纯节点拉取与 memoization 语义。
- 为每个输出建立 run provenance/data lineage。
- 为平台 adapter 建立 capability discovery 与 eligibility 诊断。

### P2：生成体验和维护效率

- 从 contract 生成 catalog、节点卡片、表单和参考文档。
- 让文档示例参与 schema 验证和 smoke test。
- 在画布上区分 unsupported、permission-required、deprecated 和 migration-required。
- 为节点错误提供稳定 code、用户说明与 remediation。
- 为 contract diff 增加 breaking-change 检查：端口删除、改型、必填变化、默认语义变化、纯度/能力变化。

## 10. 可验证的验收标准

一次节点架构升级至少应通过以下契约测试：

1. `pure-data` contract 的 exec pins 必须为空，画布也渲染为零个执行端口。
2. contract 中的每个 pin 在后端、序列化、前端和文档中的稳定 ID 一致。
3. 任意前端 fallback 都不能创造 contract 未声明的 pin。
4. 类型不兼容 edge 在拖线、加载和执行前得到相同诊断 code。
5. 已保存 v1 节点在发布 v2 后仍按 v1 运行，或经过显式 migration 后按 v2 运行。
6. 缺少平台 capability 的节点不会进入执行，并返回具体缺失项。
7. 未授权 effect 不调用 adapter。
8. retry 不会重复已成功的不可幂等 effect，或系统明确禁止 retry。
9. 相同输入的 deterministic pure graph 可重复得到相同结果与依赖顺序。
10. replay 不会重新调用已记录的 AI、网络、时间或随机 effect。
11. 错误能定位到 run、node、port/item 和 attempt，并保留 cause。
12. 每个 catalog 节点页面由 contract 生成，列出输入、输出、错误、平台、能力、版本与示例。
13. schema `default` 的应用由 Yotta 显式实现并测试，不依赖 JSON Schema validator 自动填值。
14. contract 示例本身通过 schema 和连接验证。

## 结论

Yotta 应把节点系统重构为“**版本化 canonical node contract + 严格分离的 exec/data graph + 确定性 scheduler + capability-bounded effect adapters + 结构化错误与数据血缘 + schema 驱动的 UI/文档**”。

其中最直接、优先级最高的规则是：**前端永远不能推测端口。**对拼接等纯数据节点，正确 contract 就是两个 data inputs、一个 data output、零个 exec inputs、零个 exec outputs。所谓“通用 `out`”若不在后端契约中，就不是兼容性功能，而是语义污染。
