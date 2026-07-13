---
topic: major-upgrade-review
title: "Yotta 3.0 major upgrade"
summary: "Implement and validate the AI-native destructive Yotta 3.0 architecture and release program."
---

## State

Yotta 3.0 全仓方案已进入实施。Wave 1 已完成并通过 Standards/Spec 双轴 review。Wave 2 工程实现已完成：精确工具链/Wails runtime、immutable Actions、frozen build、Windows staging/manifest、SBOM/checksum/attestation candidate、CodeQL/dependency review/secret scan/Scorecard 与双 clean-build 比较均已落地；公开 stable 仍被 owner 设置、代码签名和用户数据目录迁移阻断。

## Next

Wave 3 第一切片已完成并通过双轴 review 修复。第二切片已建立 RFC 8785/domain-separated content identity、machine-only CatalogSnapshot、受预算约束的 static compiler core 与可信重绑定的 opaque ProgramSnapshot；第三切片已加入 compiler-owned typed subgraph interface、静态 call closure/cycle rejection、reachable-only locks/capabilities 与冻结 call plan；第四切片已用 Catalog v2 声明式 dynamic descriptor 取代三个动态布尔契约，并在 Program v3 中严格冻结 Switch case outputs；第五切片已把可判定的输入约束纳入 Catalog v3 identity、编译诊断与 runtime adapter。下一步继续收敛 declarative dependency/effect 与剩余 custom validation，完成前仍禁止 Runtime 接入。项目所有者仍须并行拍板 Wave 0 的 OSI license、canonical identity 和公开主线，未完成前不能发布 Source Open/Stable。

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
- work/major-upgrade-review/research/oss-platforms.md — 需要复核 n8n/Node-RED/Windmill/Temporal/VS Code/ComfyUI 取舍时
- work/major-upgrade-review/research/ai-prompting.md — 需要复核最新模型、provider、prompt/tool/schema/eval/MCP 决策时
- work/major-upgrade-review/research/oss-governance.md — 需要执行 license、ruleset、release、SLSA/OpenSSF 路线时

## Progress

Done:
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
- Program v3 把每个节点的 resolved dynamic ports 冻结为必填非 nil 计划；`OpenProgram` 从 trusted Catalog 与 config 重派生并 exact compare，伪造 plan/config 均 fail closed。
- Standards/Spec/Threat 终审修复 descriptor 泛型消费者仍读固定 config key、前端重复 parser，以及遗漏 U+061C bidi control 的动态端口名过滤；复核后无剩余 P0–P2。
- 新增 `InputSpec.Constraints` 的 `nonBlank` 与 `numberGreaterThan` 声明式约束；Registry 统一校验和 canonicalize，Catalog v3 冻结约束，Compiler 对静态值/默认值产出稳定诊断并为全图评估设预算，接线值延迟到 runtime 校验。

Current:
- Wave 2、Wave 3 strict Source、static compiler core、typed subgraph closure、declarative Switch dynamic contract 与首个 declarative input constraint 切片已完成。下一入口是 dependency/effect 与剩余 custom validation；在这些完成前禁止 Runtime 接入或把 compiler core 宣称为完整 compiler。

Verified:
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

## Open questions

- OSI 许可证由权利人选择；方案默认建议 Apache-2.0。
- canonical GitHub org/repo 是否确定为 `yottaapp/yotta`，以及如何把本地领先历史安全公开。
- Wave 0 的法律与远端治理项应由 owner 并行处理；工程主线下一入口固定为 Wave 3。
- 是否把完整路线拆成 issue；插件门 C 明确不属于 3.0 stable 的必交付范围。
- 本机全局 Node 仍是 22.14；engine-strict 会正确拒绝，Wave 2 验证使用经官方 SHA256 校验的临时 Node 22.23.1。开发机应升级全局 Node。
- stable installer 仍缺 Yotta/capture Authenticode 签名，且应用当前把 settings/data/logs 写到 exe 旁；迁移到用户可写目录前不得恢复安装器发布。
- GitHub rulesets、push protection、private vulnerability reporting、immutable releases 与 release environment 审批需要 owner 在远端启用并验证。
- editor 距最终 450 KB target 还差约 19 KB，进入后续 bundle 优化；完整 Tabler 数据已不再打包。

