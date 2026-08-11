# V4 文档与知识刷新

## Goal

以当前代码、schema、Task、测试和生成合同为唯一事实依据，全面复核并重写 Yotta 的公开 README、`docs/`
系统文档和 `flightdeck/knowledge/` 任务指南；删除旧 YHBox 预览资产，补齐当前 V4 产品、运行、数据、Target、
资源、调试、发布与开发路径中缺失的说明，并建立可重复的文档真实性检查。

## Status

Finished

## Current

已按生产代码/schema/Task/测试/生成契约逐篇复核 `docs/` 全部 13 篇；每页 evidence/result 位于
`references/docs-audit.md`。此前遗漏的 `open-source-readiness.md` 已整体重写，并修正 Application 实例边界、
Compiler/PreviewRun、Settings commit、Run durable facts、Schedule dispatch/once、CLI flag/migration、platform
guest isolation 及根 README/CONTEXT 等实质漂移。`contracts.md`、`storage.md` 是复核后保留，不是用门禁替代
语义审计。

最终 `task check` 退出码 0，用时 300.2 秒：94 个受影响 Go package、前端 110 个 test file/486 项测试，以及
docs、contract、version、bindings、AI eval 和供应链检查全部通过。第一次门禁暴露 observation integration 的
20ms 真实墙钟 flake；将测试预算与 deadline 单测职责分离后，定向集成/时钟测试各连续 100 次、原 package 和
完整门禁均通过。当前 remote 仍是旧项目地址，Schedule once/Reload 行为仍需独立产品决策；本 Work 未修改
remote、历史、tag 或发布状态。

## Next

None

## Progress

- 2026-08-11 建立代码优先的 V4 文档刷新 Work；确认无匹配 Open Work，现有 Knowledge 与 docs 全部重新
  接受实现核验，不继承旧整合工作的正确性结论。
- 2026-08-11 从 `internal/desktopapp`、`localruntime`、Workflow schema/compiler/Application、services、
  automation adapters、storage/catalog、frontend routes、Taskfile、CI、release scripts 与生成 RPC/Node
  contracts 建立当前事实清单；`task versions:inventory` 成功输出所有独立版本域。
- 2026-08-11 删除 owner 明确授权的 `preview/fish.png` 与 `preview/piano.png`；未触碰其它旧 fixture 或 smoke。
- 2026-08-11 重写根 README 与现有系统文档，新增 Workflow、Target/Resource、Run/Schedule、CLI 和 contract
  页面；修正 Snippet、Schedule、MCP/AI 执行边界、Application Run 顺序、平台支持与存储/迁移等漂移。
- 2026-08-11 重写 Knowledge 导航及 automation、build、frontend、Wails 指南，新增 runtime 配置生命周期、
  storage migration 与文档维护指南；并在根 `AGENTS.md` 固化“文档/Knowledge 只导航、必须回查实现”的
  规则，所有结论均不以旧 Knowledge 自证。
- 2026-08-11 新增 `check:docs` 及 changed-file 路由；校验 40 份稳定 Markdown 的本地链接、Task 名和禁用旧
  公开引用。`task check:changed:self`、`task check:docs`、`git diff --check` 和 `task check` 均成功；最后一次
  `task check` 用时 227.9 秒，Go 94 个受影响包与前端 110 个测试文件/486 项测试通过。
- 2026-08-11 Owner 复核后发现逐篇语义审计证据不足；确认 `open-source-readiness.md` 为遗漏页，重新打开
  Work 并撤回完成结论，改为对 `docs/` 当前全部 13 篇建立显式覆盖。
- 2026-08-11 完成 13 篇 claim-to-code 审计并新增 `references/docs-audit.md`。重写发布就绪与兼容迁移页，
  修正 runtime/Target/Run/Schedule/CLI/platform 等语义；同时修正根 README/CONTEXT 和误导代码注释。审计还
  发现 Schedule `once` 会在每次 daemon Reload 重新触发，已按当前代码记录为待 owner 决定的行为。
- 2026-08-11 `task check` 首次运行在 observation integration 的 20ms wall-clock 断言失败；目标 test 单独
  20 次不复现，结合 fake-clock deadline tests 证明是负载 flake。提高 integration fixture 的 timeout、保留精确
  capture/journal 断言后，集成与 deadline tests 各 100 次、`internal/noderuntime` package 和最终 `task check`
  全部通过；最终完整门禁用时 300.2 秒。
