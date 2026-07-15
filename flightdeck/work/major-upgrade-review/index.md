---
topic: major-upgrade-review
title: "Yotta 3.1 major upgrade"
summary: "Implement and validate the AI-native destructive Yotta 3.1 architecture and release program."
---

## State

Yotta 3.1 contract kernel、首条 Concat tracer，以及 Blob/Stream/Resource kernel 已落地。ValueEnvelope 现在按 pinned Data Type 封闭为 inline/blob/stream/handle 四分支；runtime authority 无 durable artifact。资产已破坏性切换到 schema v2 typed BlobRef、严格 preload、引用完整性验证和排斥 GC 的 commit API；旧 BlobStore、string SHA 与整 Blob thumbnail RPC 已删除。Resource Broker 把 Program/plan/principal/plugin/session/Run/invocation/provider/target/operation/expiry 绑定为唯一 lease authority，并对 open/borrow/call 逐次授权。3.1 interpreter 仍是 fail-closed pure-data preview；conversion nodes、正式 Program/Run composition 与 plugin host conformance 未完成前不得接成生产 fallback。

## Next

下一纵向切片先定义 Program/Run artifact 与 composition owner，把已完成的 Broker/Stream kernel 接入真实 Run grant、终止撤销和 trace redaction；同时用一个显式 blob↔stream conversion tracer 验证 effect/capability/error contract。随后进入 Authoring Projection 和 built-in catalog 批量迁移。Program/Run 决议和 catalog-wide 编译完成前禁止把 3.1 interpreter 接成生产 fallback。
## Read now

- knowledge/agent/codex-working-agreement.md
- work/major-upgrade-review/context.md
- work/major-upgrade-review/review.md
- work/major-upgrade-review/design.md
- work/major-upgrade-review/ai-native-design.md
- work/major-upgrade-review/plan.md
- knowledge/build/ci-documented-gates-can-be-absent.md
- knowledge/build/wails-rpc-count-is-not-a-contract.md
- knowledge/mcp/normalize-masks-schema-prompt-drift.md

## Read if

- knowledge/build/build.md — 开始运行构建、测试或产物验证前
- knowledge/architecture/go-module-identity.md — 方案涉及 module、仓库身份或 bindings 路径时
- knowledge/architecture/go-multiplatform-boundary.md — 评估跨平台 seam 与发布声明时
- knowledge/architecture/content-addressed-workflow-artifacts.md — 修改 Source/Catalog/Compiler/Program identity 或执行绑定时
- knowledge/agent/untracked-agent-instructions-drift.md — 新建 tracked AGENTS 或调整 provider-specific agent 指令时
- knowledge/flightdeck/checkpoint-large-document-truncation.md — checkpoint 包含经 shell 分页读取的大型 topic/research 文档时
- work/major-upgrade-review/research/oss-platforms.md — 需要复核 n8n/Node-RED/Windmill/Temporal/VS Code/ComfyUI 取舍时
- work/major-upgrade-review/research/ai-prompting.md — 需要复核最新模型、provider、prompt/tool/schema/eval/MCP 决策时
- work/major-upgrade-review/research/oss-governance.md — 需要执行 license、ruleset、release、SLSA/OpenSSF 路线时

## Progress

Done:
- 关闭 Capability/Target Planning Wayfinder gate：Host Profile、Automation Target、Capability Definition/Requirement/Plan、Run Grant 与 Credential Binding 分离；授权同时绑定 Program plan 与 Run，不允许 ambient ServiceBundle 或 token passthrough。
- 完成 Node Contract 3.1 opaque seal/open、generated JSON Schema/TypeScript 与 semantic/authoring digest 分离；端口按 data/exec/error/status 明确分频道。
- 完成 Catalog 3.1 exact TypeRef/NodeRef/implementation lock snapshot，并删除旧 `internal/workflow/catalog` 投影。
- 破坏性切换 Workflow Source/Compiler/Program 到 3.1：显式 edge channel、typed endpoint、literal/default provenance、strict trusted reopen、全边界资源预算与 fail-closed preview feature set。
- 新增 ValueEnvelope 3.1；interpreter 使用可信 Catalog、capability grant 和完整 installed implementation lock，输出按 pinned Data Type 复验并限制 retained bytes。
- 完成 Concat Source → Compiler → Program → interpreter → Vue/MCP/docs tracer；Vue 未知 pin 不再猜成 exec，generated machine Catalog 可由 strict opener 原样打开。
- 新增 immutable Blob Store：raw-byte SHA-256、typed media/size reference、单体/总量 quota、range read、读前 integrity、dedup 复验、ownership marker、durable replace/remove 与 stop-the-world Sweep。
- 资产 schema 破坏性升至 v2：删除 string SHA/旧 BlobStore/skip-corrupt reader；record/variant 只能经原子 blob-reference commit API 引入，preload 严格拒绝旧版、未知字段、重复字段、越界 JSON、意外目录项和 dangling/tampered blob。
- 新增 Run-scoped Resource Broker 与 Stream provider：256-bit opaque token、完整 authority scope、逐次 open/borrow/call authorize、narrow lease、expiry/Run revoke/Broker close、active cancellation、exactly-once cleanup、bounded backpressure、finish drain/EOF 与 cancel wakeup。
- ValueEnvelope 四分支进入 v2 digest preimage 并由 pinned Data Type representation/codec/schema 复验；inline 上限 1 MiB，stream/handle 只提供显式 RuntimeArtifact，Durable Artifact 恒为 nil。
- 删除 `ReadBlobDataURL` Wails RPC 及前端 thumbnail base64 fallback；前端只接收 typed BlobRef 元数据，bounded preview adapter 完成前使用明确 placeholder。
- Standards/Spec 双轴复审发现的 implementation bypass、untyped output、silent control semantics、Source semantic drop、fake exec-out、canonical catalog 与 Program strict-boundary 问题均已修复并加回归测试。
- 落地 Data Type 3.1 Contract Kernel：opaque definition seal/open、算法域 `/v1` semantic digest、版本化 TypeRef、离线 Draft 2020-12 bundle 与真实 schema 引用预算、codec/editor allowlist、Resolved Type、union/list assignability；双轴终审无剩余 P1/P2。
- 完成 Node System 3.1 设计变更的 Standards/Spec 双轴 review；修正 Data Type semantic digest 自引用、list 运行类型身份与研究/决议漂移。
- 将插件安装信任、SDK/文档/conformance 从 Fog 升格为阻塞 tickets，并明确 Yotta 3.1 产品版本与 3.1 协议代际可合并且必须共用唯一 runtime。
- 盘点 1,169 个 tracked files 与主要 package/module。
- 审查应用装配、节点/运行时、持久化、前端 contract/editor、CI、供应链与开源治理。
- 完成 `review.md` 成熟度评分与 P0/P1 缺口，保留/替换/延期边界明确。
- 完成 `design.md` 共享核心与 `ai-native-design.md` 产品/AI 目标架构。
- 将旧九阶段工程计划重排为 13 个 Wave、5 个 release milestone 与 23 个推荐纵向 PR。
- 完成本地 AI/LLM/MCP prompt surface 审计：当前 provider 是 Chat 最低公分母、结构化输出存在 prompt fallback、模型能力靠 endpoint 猜测、节点目录一次返回约 13.3 万字符。
- 确认 MCP graph 示例与真实 schema 漂移被 `Normalize()` 掩盖。
- 完成 n8n、Node-RED、Windmill、Temporal、VS Code、ComfyUI 一手资料对标，共 63 个官方来源。
- 完成 OpenAI、Anthropic、MCP prompt/tool/eval/安全研究，保留 78 处直接官方链接（53 个独立 URL）。
- 完成 OSI/OpenSSF/SLSA/GitHub/DCO/CNCF 治理、供应链与发布研究。
- 完成 188 个前端文件的一次性 oxfmt 基线；format/typecheck/i18n/68 files 529 tests 全绿。
- 新增 tracked `AGENTS.md` 与 `task check`，README/CONTRIBUTING 不再维护平行的全套命令清单。
- 将 oxlint/eslint 改为 check-only；清除真实 lint 错误，276 个 `no-explicit-any` 用精确 baseline ratchet，增减均需显式审查。
- Go coverage 从 64.3% 提升到 65.3%，全局 floor 65%；root/appruntime/MCP/recording/capture/input 另有 package floor。
- 用 `contracts/wails-rpc.json` 取代失真的 `14 Services, 107 Methods` CI 字符串；当前真实 contract 为 14 services / 112 methods / 86 models+enums。
- 新增 bounded fuzz smoke：graph rewrite、package metadata、MCP node params、expression parser。
- 新增 Vite manifest bundle gate：entry 308,104 / 350,000 bytes；editor 初始同步闭包 468,811 / 650,000 bytes，最终目标 450,000；完整 Tabler `icons.json` 被 forbidden check 禁止。
- ELK 改为首次自动布局时动态加载，editor 初始同步 gzip 由约 905 KB 降至 468 KB。
- 图标搜索改为构建期提取、运行时懒加载名称索引：约 21,825 bytes gzip，取代 331,580 bytes 的完整图标数据 chunk。
- Windows required quality job 直接运行 `task check`；Linux/macOS/Windows production GUI compile matrix 全部生成 bindings 与前端产物。
- tracked `CLAUDE.md` 已收敛为只引用 `AGENTS.md`/Flightdeck 的薄 wrapper，旧 `YHFish`、HARD-GATE 与直推 main 指令已删除。
- review 修复 Switch case 删除的非原子历史写入，以及 ELK lazy-load/layout 期间切图后写错 graph/marker 的竞态。
- 删除 NodeSpec 的 `DynamicOutputs` / `DynamicInputs` / `DynamicDataFields` 三个行为布尔值，改为进入 Catalog v2 identity 的 role/shape/config-key/budget descriptor；旧运行时消费者与前端均从 descriptor 派生。
- 新 Compiler 首个动态切片严格支持 Switch `output + names + Exec`：拒绝错 shape、非字符串、空白/点/control/bidi、重复、静态冲突和预算放大，并隔离不同节点的 resolved pin index。
- 历史 v3 prototype 曾冻结 resolved dynamic ports；该未发布 wire contract 已由 Program 3.1 destructive cutover 删除，后续动态端口必须以 3.1 Node Contract/Program 决议重新进入。
- Standards/Spec/Threat 终审修复 descriptor 泛型消费者仍读固定 config key、前端重复 parser，以及遗漏 U+061C bidi control 的动态端口名过滤；复核后无剩余 P0–P2。
- 历史 prototype 新增 `InputSpec.Constraints` 的声明式约束；旧 Catalog v3 投影已删除，约束在 3.1 的最终表达仍由 Data Type/Node Contract 后续切片决定。
- 截图新模板表单已加入可创建的分类选择，`SaveTemplateCapture` 在首次资产记录中原子保存 category；主窗口把历史小宽度钳到 1640 并设置系统最小宽度，短期阻止编辑器工具条在低宽度下崩坏。
- 容器 Windows 输入缺省已从 PostMessage 改为 SendInput：新建容器、旧记录空字段、运行时 backend 构造与置前判断统一走前台默认；显式 PostMessage 保持不变。模板缩放容差 UI 改为最大倍率并实时解释 `[1/k,k]` 范围。

Current:
- Blob/Stream/Resource kernel 与资产 destructive cutover 已完成并通过双轴 review；resource ticket 保持 open，明确阻塞项只剩 conversion nodes、正式 Program/Run composition、三 host conformance 和 capture/preview transport。当前代码保持 pure-data fail closed，不建立 resource/effect runtime 旁路。

Verified:
- Blob/Stream/Resource 与 asset schema v2 批次 `task check` 通过（2026-07-15，174.4s）：全局 coverage 65.6%，frontend 97 files / 635 tests，Wails contract 14 services / 118 methods / 102 models；聚焦 blob/resource/stream/asset race 全绿。
- 3.1 Concat tracer destructive cutover 最终 `task check` 通过（2026-07-15，117s）：全局 coverage 65.2%，frontend 97 files / 635 tests，Wails contract 14 services / 119 methods / 100 models，contracts drift/staticcheck/vet/build/bundle budget 全绿。
- `go test -race` 覆盖 artifact/datatype/nodecatalog/nodecontract/nodes31/workflow schema/compiler；ParseSource、CompileDraft、OpenProgram fuzz 各 5 秒通过（约 10.4 万、17.5 万、36.6 万 executions）。
- Data Type 3.1 kernel 最终 `task check` 通过（2026-07-15，109.9s）；聚焦 race、5 秒 fuzz、vet、staticcheck 通过，datatype coverage 71.0%。Standards/Spec 复核确认无剩余 P1/P2。
- Node System 3.1 与 Yotta 3.1 计划合并后，Flightdeck recovery graph 无诊断，Wayfinder 10 个 tickets 依赖闭合且唯一 frontier 为 Node Contract 3.1 元模式；完整 `task check` 通过（2026-07-15，107.6s）。
- 用户已明确允许破坏性升级，不要求兼容与兜底。
- `go test ./...`、`go vet ./...`、`staticcheck ./...` 通过。
- frontend Vitest 70 files / 537 tests、vue-tsc、i18n、format/lint 与 production build 通过。
- 三份外部研究均只采用项目官方文档、官方仓库或官方规范，并已保留直接链接。
- `task check:frontend` 全绿：frozen install、bindings generation/contract、format、oxlint、eslint baseline、vue-tsc、i18n、70 files / 537 tests、production build、bundle budget。
- `task check:go` 全绿：全量 atomic coverage 65.3%、关键包 floors、vet、staticcheck。
- Wave 3 strict parser 由生成的 Draft 2020-12 Schema 直接驱动，结构规则不再手写复制；schema 包 coverage 77.2%，parser fuzz 3 秒完成 80,751 次执行。
- `task check:fuzz FUZZ_TIME=2s` 七个 fuzz target 全绿（含 strict Source、CompileDraft 与 trusted Program open）；CI 配置为各 10 秒。
- `go test -race -count=1 ./internal/node ./cmd/node-catalog` 通过。
- Switch dynamic contract 切片在终审修复后完整 `task check` 通过：Go atomic coverage 66.4%（floor 65%），Wails contract 14 services / 112 methods / 89 models，frontend 70 files / 537 tests，entry 309,160 / 350,000 bytes、editor 468,818 / 650,000 bytes。
- `go test -race -count=1 ./internal/workflow/schema ./internal/workflow/catalog ./internal/workflow/compiler` 通过；ParseSource、CompileDraft、OpenProgram 三个 fuzz target 各运行 5 秒通过。
- declarative input constraints 聚焦门禁通过：`go test ./internal/node ./internal/nodes/system ./internal/workflow/catalog ./internal/workflow/compiler ./internal/workflow/schema`。
- 模板分类与主窗口宽度修复后完整 `task check` 通过：Go coverage 66.2%，frontend 71 files / 538 tests，Wails contract 14 services / 112 methods / 91 models，entry 309,295 / 350,000 bytes，editor 471,230 / 650,000 bytes。
- 输入默认与缩放容差说明更新后完整 `task check` 通过：Go coverage 66.5%，frontend 89 files / 590 tests，Wails contract 14 services / 116 methods / 100 models，entry 323,845 / 350,000 bytes，editor 462,559 / 650,000 bytes。

## Open questions

- Program/Run 的 durable Run/trace/error artifact、Broker composition owner、外部 capability grant 生命周期和 package trust 仍由对应 Wayfinder tickets 决定；preview 必须继续 fail closed。
- OSI 许可证由权利人选择；方案默认建议 Apache-2.0。
- canonical GitHub org/repo 是否确定为 `yottaapp/yotta`，以及如何把本地领先历史安全公开。
- Wave 0 的法律与远端治理项应由 owner 并行处理；工程主线下一入口固定为 Wave 3。
- 完整路线已由 Node System 3.1 Wayfinder tickets 表达；最小 Wasm + Process Plugin Host 已纳入 3.1 stable，marketplace 仍不在范围内。
- 本机全局 Node 仍是 22.14；engine-strict 会正确拒绝，Wave 2 验证使用经官方 SHA256 校验的临时 Node 22.23.1。开发机应升级全局 Node。
- stable installer 仍缺 Yotta/capture Authenticode 签名，且应用当前把 settings/data/logs 写到 exe 旁；迁移到用户可写目录前不得恢复安装器发布。
- 编辑器 UI 需要结构性升级而非换皮：1640 最小宽度只是短期 containment；后续应优先做画布空间预算、面板互斥/overlay、紧凑上下文工具条、Basic/Pro 渐进披露与可恢复错误。
- GitHub rulesets、push protection、private vulnerability reporting、immutable releases 与 release environment 审批需要 owner 在远端启用并验证。
- editor 距最终 450 KB target 还差约 19 KB，进入后续 bundle 优化；完整 Tabler 数据已不再打包。
