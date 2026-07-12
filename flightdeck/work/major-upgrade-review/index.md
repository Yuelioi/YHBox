# Index — major-upgrade-review

## State

Yotta 3.0 全仓审查与 AI-native 大型开源升级方案已完成。方案统一覆盖法律/公开主线、架构、工作流编译、执行可靠性、前端、Node SDK、权限、provider/prompt/eval、MCP、插件、供应链与社区治理。

## Next

由用户确认实施入口。推荐先处理 Wave 0 的 OSI license、canonical identity、公开主线/ruleset 和 stable release freeze；若法律决策需要等待，可并行开始 Wave 1 的格式基线与真实 CI gate，但不能跳过 Wave 0 发布 Source Open/Stable。

## Read now

- `flightdeck/knowledge/agent/codex-working-agreement.md`
- `flightdeck/work/major-upgrade-review/review.md`
- `flightdeck/work/major-upgrade-review/design.md`
- `flightdeck/work/major-upgrade-review/ai-native-design.md`
- `flightdeck/work/major-upgrade-review/plan.md`
- `flightdeck/knowledge/build/ci-documented-gates-can-be-absent.md`
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

Current:
- 方案已完成，等待用户选择是否进入 Wave 0/1 实施，或先把路线拆为可独立领取的 issue。

Verified:
- 用户已明确允许破坏性升级，不要求兼容与兜底。
- `go test ./...`、`go vet ./...`、`staticcheck ./...` 通过。
- frontend Vitest 68 files / 529 tests、vue-tsc、i18n 通过。
- frontend production build 通过；editor chunk 2.74 MB（gzip 858.56 KB）。
- frontend format check 失败：188 个文件需要格式化；CI 当前未运行该门禁。
- 三份外部研究均只采用项目官方文档、官方仓库或官方规范，并已保留直接链接。
- 文档 diff 通过 `git diff --check`；本批只改文档，未重复运行代码测试。

## Open questions

- OSI 许可证由权利人选择；方案默认建议 Apache-2.0。
- canonical GitHub org/repo 是否确定为 `yottaapp/yotta`，以及如何把本地领先历史安全公开。
- 是否立即进入 Wave 0；若先做无需法律拍板的工程工作，入口固定为 Wave 1 的格式基线与真实 CI gate。
- 是否把完整路线拆成 issue；插件门 C 明确不属于 3.0 stable 的必交付范围。
