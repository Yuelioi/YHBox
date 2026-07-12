# Index — major-upgrade-review

## State

原 Yotta 3.0 工程方案已完成；正在扩展为 AI-native 大型开源项目方案，并行对标优秀项目、最新 prompt/tool/eval 实践与开源治理。

## Next

完成三条外部一手资料研究，合并本地 AI/MCP 审计，新增 AI-native 目标设计并重排实施优先级。

## Read now

- `flightdeck/knowledge/agent/codex-working-agreement.md`
- `flightdeck/work/major-upgrade-review/design.md`
- `flightdeck/work/major-upgrade-review/plan.md`
- `flightdeck/knowledge/build/ci-documented-gates-can-be-absent.md`
- `flightdeck/knowledge/mcp/normalize-masks-schema-prompt-drift.md`

## Read if

- `flightdeck/knowledge/build/build.md` — 开始运行构建、测试或产物验证前
- `flightdeck/knowledge/architecture/go-module-identity.md` — 方案涉及 module、仓库身份或 bindings 路径时
- `flightdeck/knowledge/architecture/go-multiplatform-boundary.md` — 评估跨平台 seam 与发布声明时

## Progress

Done:
- 盘点 1,169 个 tracked files 与主要 package/module。
- 审查应用装配、节点/运行时、持久化、前端 contract/editor、CI、供应链与开源治理。
- 完成 `design.md` 目标架构与 `plan.md` 九阶段实施方案。
- 完成本地 AI/LLM/MCP prompt surface 审计：当前 provider 是 Chat 最低公分母、结构化输出存在 prompt fallback、模型能力靠 endpoint 猜测、节点目录一次返回约 13.3 万字符。
- 确认 MCP graph 示例与真实 schema 漂移被 `Normalize()` 掩盖。

Current:
- 外部工作流平台、AI prompting、开源治理三条一手资料调研。

Verified:
- 用户已明确允许破坏性升级，不要求兼容与兜底。
- `go test ./...`、`go vet ./...`、`staticcheck ./...` 通过。
- frontend Vitest 68 files / 529 tests、vue-tsc、i18n 通过。
- frontend production build 通过；editor chunk 2.74 MB（gzip 858.56 KB）。
- frontend format check 失败：188 个文件需要格式化；CI 当前未运行该门禁。

## Open questions

- OSI 许可证由权利人选择；方案默认建议 Apache-2.0。
- 是否立即进入实施，以及从 Phase 0 还是 Phase 1 开始。
- AI-native 方案完成后需重新确定 Phase 顺序；当前不再建议直接按旧九阶段开始编码。
