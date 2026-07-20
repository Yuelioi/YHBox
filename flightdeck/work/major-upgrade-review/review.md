# Yotta 全仓升级审查结论

## 执行摘要

Yotta 不是需要推倒重写的原型。它已经具备大型产品的功能广度和一部分可靠性基础：Go 测试规模可观，portable core、原子持久化、节点 capability 校验、应用生命周期 owner、前端组件测试与中英文产品面都已存在。

真正的问题是：**产品能力增长得比内部契约、公开治理和发布可信度快。** 当前最危险的不是某个函数写得不好，而是多个“事实源”同时存在：

- 内存草稿与按 ID 读取的磁盘 workflow；
- Go model、手写 TS DTO、Wails bindings、MCP 示例和 normalize 规则；
- 节点 UI 文案与给模型的 machine catalog；
- 通用 Chat 接口与 provider 的真实能力；
- 文档中的 gate 与 CI 实际执行的 gate；
- 本地领先主线与公开 GitHub 主线；
- “开源项目”定位与非 OSI LICENSE。

因此 3.1 的主任务不是再加功能，而是把 Source → Compiler → ProgramSnapshot → RunRecord 建成唯一真相链，并让 UI、AI、CLI、MCP、插件和 release 都围绕它工作。

## 当前成熟度评分

评分是本轮审查的诊断量表（0=缺失，5=大型成熟项目），不是行业认证。

| 维度 | 当前 | 3.1 目标 | 主要依据 |
| --- | ---: | ---: | --- |
| 产品/节点能力 | 4.0 | 4.5 | 137 个节点、桌面/ADB/视觉/LLM/录制/调度能力完整 |
| 核心正确性 | 3.3 | 4.5 | Go 测试与原子存储良好；运行快照、持久 Run 事实仍缺失 |
| 模块架构 | 2.2 | 4.2 | root 装配约 2,500 行、后置注入、node 反向依赖 services、宽 ServiceBundle |
| 跨层契约 | 1.8 | 4.6 | Go/TS/MCP/示例平行定义，Normalize 会掩盖漂移 |
| 前端可维护性 | 2.2 | 4.2 | 529 tests；两个 1,200–1,650 行组件、手写 DTO 与巨大 editor bundle |
| AI/model readiness | 1.5 | 4.5 | Chat 最低公分母、JSON prompt fallback、弱 schema、无版本化 eval/trace |
| 扩展作者体验 | 2.0 | 4.0 | 节点基础丰富；缺单一 NodeSpec 生成链、SDK/lint/fake capability |
| 安全/权限 | 2.0 | 4.3 | 有 arm 与部分脱敏；MCP 默认监听、key 明文、缺统一 capability policy |
| CI/质量门禁 | 2.3 | 4.5 | 本地 test/typecheck/build 可过；CI 未执行全部前端 gate，coverage/bundle 无阈值 |
| 供应链/release | 1.2 | 4.5 | 可变依赖/Action、裸 unsigned exe、无 SBOM/checksum/provenance |
| 开源治理 | 0.8 | 4.0 | 非 OSI LICENSE、身份/公开主线分裂、无 ruleset/治理、bus factor 1 |

## 应该保留并深化的资产

1. **Go 作为核心运行时。** 不需要为“现代化”改语言；应把 Compiler、Program、Workspace、RunStore 做成深模块。
2. **Wails + Vue 产品形态。** 桌面交互是 Yotta 的差异化；问题是契约与状态边界，不是框架本身。
3. **单机串行 Worker。** 键鼠与目标窗口具有排他性；不应照搬云工作流的分布式队列。
4. **现有 portable-core 和 adapter 方向。** 跨平台 compile gate、平台 adapter 与生命周期 owner 是可继续深化的正确基础。
5. **节点库与录制能力。** 这是项目护城河；3.1 应优先统一 Spec、effect 与生成工具，而不是重写每个算法。
6. **原子文件写、action trace 与已有安全校验。** 它们应被 Workspace transaction、Run timeline 与统一 redaction 吸收。
7. **已有 `AISlot` 概念。** 它适合作为可移植模型意图，不应让节点继续绑定具体 connection/model string。

## 必须删除或替换的结构

### P0：发布/法律阻断

- 非 OSI LICENSE 与“开源”表述冲突；canonical repo/module/org/update/security identity 分裂。
- GitHub main/tag 缺少强制保护，只有一个管理员；公开主线落后本地实际开发历史。
- 可变 Actions/toolchain 与未签名裸 exe release；缺 checksum、SBOM、provenance、immutable release。

这些不是文档小修。未处理前，不应对外宣称“大型开源项目”，也不应再发 stable tag。

### P0：正确性阻断

- `ValidateContainerByID` 让未保存草稿必须先落盘；执行也缺少锁定的 immutable program。
- 内存 queue/递增 ID 是唯一运行状态；崩溃后无法可靠区分取消、中断和已产生的副作用。
- MCP 示例使用错误 schema 字段，却被测试先 `Normalize()` 修复，形成假绿。
- whole-container JSON 的 MCP save/validate 会把模型失误放大为整图覆盖。

### P1：架构债

- `main.go` 与 `wire_*.go` 构成巨大 composition root，constructor 后再 Configure/Set 依赖。
- `internal/node` 知道 concrete services；`ServiceBundle` 与 `VisionService` 是宽接口。
- global registry/blank import/`init()` 隐藏生产 catalog；测试与生产装配不对称。
- container/subgraph/asset/blob 的多步写入靠多个 owner 和 callback 拼接。

### P1：AI 协议债

- `Provider.Chat` 把不同 provider 压成 messages → text，无法可靠表达 tool/usage/reasoning/stream/result。
- `ModeAuto` 通过 endpoint 猜能力；`structuredViaPrompt` 用“ONLY JSON”+字符串截取模拟协议。
- schema 只能表达扁平 name/type；动态模板可进入 system 指令；缺字段时可能保留旧变量。
- API key 作为普通 settings 明文持久化并返回前端。
- 137 节点 catalog 一次约 13.3 万字符，模型发现工具的上下文设计失效。

### P1：维护与性能债

- `ContainerEditorView.vue` 1,652 行、`NodeInspector.vue` 1,207 行；view 同时拥有状态机、RPC、业务规则和渲染。
- `backend.ts` 手写 DTO/RPC 门面，Wails bindings 又是另一来源；pin compatibility 前后端平行实现。
- editor chunk 2.74 MB（gzip 858.56 KB），icons chunk 2.10 MB，无机器预算。
- `format:check` 当前失败 188 个文件；CI 未运行 Vitest/typecheck/i18n/format/build 全套门禁。
- 旧 provider-specific `CLAUDE.md` 已删除；仓库级 agent contract 由受版本控制的 `AGENTS.md` 单点持有。

## 外部项目给出的边界，而非功能清单

| 项目/标准 | Yotta 采用 | Yotta 明确不采用 |
| --- | --- | --- |
| n8n | workflow history 与 run history 分离、dirty state、节点工具链 | 长期节点旧版本分支、Redis 队列、未隔离社区代码 |
| Node-RED | Runtime/Editor/API seam、revision conflict、轻节点包 | Runtime/UI 双份节点契约、两代 flow API |
| Windmill | immutable program hash、Run/step attempt、指标 | 每节点数据库 job、任意多语言 runtime |
| Temporal | deterministic control/effect 边界、声明式 retry 思维 | 透明重放桌面副作用、Temporal 集群 |
| VS Code | contribution manifest、lazy activation、Trust、实验 API | 把 extension host 当 sandbox、插件改 Vue DOM |
| ComfyUI | 入队冻结快照、只缓存 pure、不可变 registry version | 进程内 Python/任意网页 JS、插件覆盖宿主校验 |
| OpenAI/Anthropic/MCP | provider-native API、strict schema、typed tools、eval/trace | lowest-common-denominator Chat、prompt JSON fallback、全量上下文 |
| OpenSSF/SLSA/GitHub | 可执行治理、固定供应链、attestation、双人控制 | 只增加徽章或文档而不改变真实 settings/workflow |

## 3.1 的核心对象

```text
WorkflowSource
  用户/AI 可编辑，带 revision；只表达意图
       |
       v
CompileResult
  稳定 diagnostics；成功时包含 immutable ProgramSnapshot
       |
       v
ProgramSnapshot
  canonical hash、catalog/compiler/node lock、permission manifest
       |
       v
RunRecord
  持久状态机、programHash、permissionGrantHash
       |
       v
NodeAttempt / AdapterAction
  effect、attempt、error code、脱敏 trace；构成用户可见时间线
```

这个对象链同时解决七个问题：未保存草稿验证、运行时漂移、AI 整图覆盖、重试安全、插件制品锁、运行审计和跨客户端契约。

## 推荐取舍

### 现在就做

- OSI/identity/ruleset/source history 决策；真实 CI 和工具链固定。
- v3 Source/Diagnostic/Program contract；明确拒绝 v2。
- Application/Compiler/RunStore/EditorSession 深模块。
- NodeSpec effect/capability 与生成链。
- Workflow Trust、SecretRef/OS credential store。
- provider-native AI、PromptManifest、strict schema、eval/trace。
- typed AI authoring 与 MCP 3.1。
- 完整、签名、可验证的 Windows stable release。

### 3.1 只打基础

- 官方 Node SDK 标记 `v1alpha1`，不承诺长期 Go ABI。
- OpenTelemetry 只做 opt-in exporter；本地 Run timeline 才是产品主面。
- Linux/macOS 保持 preview，直到真实 smoke、签名和权限 UX 完成。
- 声明式模板/文档扩展可设计，但不加载第三方执行代码。

### 明确延期

- Redis/PostgreSQL 分布式 queue、多人协同服务、Temporal 等外部编排平台。
- 任意 Python/JS/多语言 execution runtime。
- 第三方进程内 Go plugin。
- 无 capability broker、制品签名/锁、资源限制的 marketplace。
- “自动选最新模型”或无 eval 的模型漂移。
- 为 v2 workflow/runtime 保留兼容解释器或迁移器。

## 最终判断

如果只重构大文件和换几段 prompt，Yotta 会变得更整洁，但不会成为大型开源项目。真正的升级顺序必须是：

1. 先让项目身份、许可证、主线、CI 和发布可信；
2. 再建立 Source/Compiler/Snapshot/Run 的唯一事实链；
3. 再让 Editor、Node SDK、权限模型围绕它收敛；
4. 最后把更强模型接到 typed authoring protocol，并以 eval、trace 和宿主权限约束它。

模型更聪明的价值，是可以把 prompt 从冗长威慑改成清晰目标、schema、工具和成功标准；它不意味着可以减少编译器、事务、权限或发布门禁。Yotta 3.1 最值得追求的不是“更自主”，而是 **更少隐式状态、更少平行协议、更强可验证性**。
