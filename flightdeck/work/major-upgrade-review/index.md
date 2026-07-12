# Index — major-upgrade-review

## State

全仓审查与 Yotta 3.0 破坏性升级方案已完成；等待用户确认是否进入实施，以及从哪个 Phase 开始。

## Next

由用户确认方案；建议先执行 Phase 0 产品/许可证决策，再以 Phase 1 真实 CI 门禁作为首个代码批次。

## Read now

- `flightdeck/knowledge/agent/codex-working-agreement.md`
- `flightdeck/work/major-upgrade-review/design.md`
- `flightdeck/work/major-upgrade-review/plan.md`
- `flightdeck/knowledge/build/ci-documented-gates-can-be-absent.md`

## Read if

- `flightdeck/knowledge/build/build.md` — 开始运行构建、测试或产物验证前
- `flightdeck/knowledge/architecture/go-module-identity.md` — 方案涉及 module、仓库身份或 bindings 路径时
- `flightdeck/knowledge/architecture/go-multiplatform-boundary.md` — 评估跨平台 seam 与发布声明时

## Progress

Done:
- 盘点 1,169 个 tracked files 与主要 package/module。
- 审查应用装配、节点/运行时、持久化、前端 contract/editor、CI、供应链与开源治理。
- 完成 `design.md` 目标架构与 `plan.md` 九阶段实施方案。

Verified:
- 用户已明确允许破坏性升级，不要求兼容与兜底。
- `go test ./...`、`go vet ./...`、`staticcheck ./...` 通过。
- frontend Vitest 68 files / 529 tests、vue-tsc、i18n 通过。
- frontend production build 通过；editor chunk 2.74 MB（gzip 858.56 KB）。
- frontend format check 失败：188 个文件需要格式化；CI 当前未运行该门禁。

## Open questions

- OSI 许可证由权利人选择；方案默认建议 Apache-2.0。
- 是否立即进入实施，以及从 Phase 0 还是 Phase 1 开始。
