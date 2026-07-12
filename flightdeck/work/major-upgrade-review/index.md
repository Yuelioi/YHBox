# Index — major-upgrade-review

## State

Yotta 3.0 全仓方案已进入实施。Wave 1 已完成并通过 Standards/Spec 双轴 review：一次性前端格式基线、tracked agent contract、canonical `task check`、check-only lint、65% Go coverage 与关键包 ratchet、Wails contract manifest、前端全套 CI、race/fuzz/三平台 GUI compile smoke 和 bundle budget 均已落地；ELK 与 Tabler 搜索索引均已从 editor 初始 chunk 按需拆出。

## Next

工程侧进入 Wave 2：完整工具链/Action SHA pin、依赖与生成物可复现、release freeze。项目所有者仍须并行拍板 Wave 0 的 OSI license、canonical identity 和公开主线，未完成前不能发布 Source Open/Stable。

## Read now

- `flightdeck/knowledge/agent/codex-working-agreement.md`
- `flightdeck/work/major-upgrade-review/review.md`
- `flightdeck/work/major-upgrade-review/design.md`
- `flightdeck/work/major-upgrade-review/ai-native-design.md`
- `flightdeck/work/major-upgrade-review/plan.md`
- `flightdeck/knowledge/build/ci-documented-gates-can-be-absent.md`
- `flightdeck/knowledge/build/wails-rpc-count-is-not-a-contract.md`
- `flightdeck/knowledge/mcp/normalize-masks-schema-prompt-drift.md`

## Read if

- `flightdeck/knowledge/build/build.md` — 开始运行构建、测试或产物验证前
- `flightdeck/knowledge/architecture/go-module-identity.md` — 方案涉及 module、仓库身份或 bindings 路径时
- `flightdeck/knowledge/architecture/go-multiplatform-boundary.md` — 评估跨平台 seam 与发布声明时
- `flightdeck/knowledge/agent/untracked-agent-instructions-drift.md` — 新建 tracked AGENTS 或调整 provider-specific agent 指令时
- `flightdeck/work/major-upgrade-review/research/oss-platforms.md` — 需要复核 n8n/Node-RED/Windmill/Temporal/VS Code/ComfyUI 取舍时
- `flightdeck/work/major-upgrade-review/research/ai-prompting.md` — 需要复核最新模型、provider、prompt/tool/schema/eval/MCP 决策时
- `flightdeck/work/major-upgrade-review/research/oss-governance.md` — 需要执行 license、ruleset、release、SLSA/OpenSSF 路线时

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

Current:
- Wave 1 已完成；下一工程入口是 Wave 2 的工具链/Action pin。

Verified:
- 用户已明确允许破坏性升级，不要求兼容与兜底。
- `go test ./...`、`go vet ./...`、`staticcheck ./...` 通过。
- frontend Vitest 70 files / 536 tests、vue-tsc、i18n、format/lint 与 production build 通过。
- 三份外部研究均只采用项目官方文档、官方仓库或官方规范，并已保留直接链接。
- `task check:frontend` 全绿：frozen install、bindings generation/contract、format、oxlint、eslint baseline、vue-tsc、i18n、70 files / 536 tests、production build、bundle budget。
- `task check:go` 全绿：全量 atomic coverage 65.3%、关键包 floors、vet、staticcheck。
- `task check:fuzz FUZZ_TIME=2s` 四个 fuzz target 全绿；CI 配置为各 10 秒。
- `go test -race -count=1 ./internal/node ./cmd/node-catalog` 通过。

## Open questions

- OSI 许可证由权利人选择；方案默认建议 Apache-2.0。
- canonical GitHub org/repo 是否确定为 `yottaapp/yotta`，以及如何把本地领先历史安全公开。
- 是否立即进入 Wave 0；若先做无需法律拍板的工程工作，入口固定为 Wave 1 的格式基线与真实 CI gate。
- 是否把完整路线拆成 issue；插件门 C 明确不属于 3.0 stable 的必交付范围。
- 本机 Node 22.14 仍可完成门禁但低于新声明的 22.18 最低版本；CI 已固定 22.18.0，本机应在下一工具链批升级。
- editor 距最终 450 KB target 还差约 19 KB，进入后续 bundle 优化；完整 Tabler 数据已不再打包。
