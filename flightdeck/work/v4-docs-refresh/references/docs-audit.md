# `docs/` semantic audit

审计日期：2026-08-11。范围是当前 `docs/` 下全部 13 篇 Markdown。每一行都从生产代码、schema、Task、测试、
生成合同或本地 Git 状态重新取证；没有用另一篇 docs/Knowledge/Work 记录证明事实。

| Page | Primary implementation evidence | Result |
| --- | --- | --- |
| `docs/README.md` | 实时文件清单；`scripts/check-docs.mjs`；各目标页实际存在性 | 导航与职责复核后保留；13 篇全部可达，明确代码/schema/Task/测试/生成合同优先 |
| `docs/architecture/README.md` | `main.go`；`internal/desktopapp/desktop.go`；`internal/localruntime/runtime.go`；`internal/architecture/platform_boundaries_test.go` | 修正“GUI/CLI/Schedule 共用同一个实例”和 presentation 完全不能组装 store/service 的过度陈述；区分相同 composition path 与不同进程实例 |
| `docs/architecture/contracts.md` | `internal/datatype`、`nodecontract`、`nodecatalog`、`nodeauthoring`、`workflow/schema`、`workflow/compiler`、`run`；`cmd/yotta-versions`；Task contract/plugin/binding tasks | 逐项复核后保留；版本域、Node identity、生成投影和 Task 名与当前实现一致 |
| `docs/architecture/runtime.md` | `internal/application/source_transition.go`、`application.go`、`run_admission.go`、`run_coordinator.go`；`internal/workflow/compiler`；`internal/run/journal.go` | 修正 Compiler 输入（Authoring Projection 不是输入）、CLI/desktop 实例关系和 status event 去向；核实 Program-before-admission、durable-queue-before-enqueue 与 restart 行为 |
| `docs/architecture/storage.md` | `internal/storage/roots*.go`、`profile.go`；`internal/storage/catalog/foundation.go`、`backup.go`；`internal/services/settings_store.go`；`internal/securestore` | 路径优先级、数据地图、single writer、SQLite/backup/settings recovery 和 secure-store 平台边界逐项复核后保留 |
| `docs/architecture/threat-model.md` | production/dev manifests；`internal/processsandbox`、`scriptengine`、`pluginhost`、`wasmrunner`、`nodepackage`、`securestore`；CI security workflows | 修正 guest isolation 的 host 范围：Windows 使用 AppContainer/LPAC/Job，非 Windows fail unavailable；Configured Target 仍保持配置即授权/per-Run direct |
| `docs/compatibility.md` | `internal/storage/migrate`；`internal/services/settings_store.go`、`schedule/store.go`、`macro/{codec,service}.go`；`internal/workflowstore/migration.go`；`internal/application/source_transition.go` | 重写当前实际迁移表；删除不存在的“3.1 settings version”说法，记录 durable rewrite/CAS、空 Source migration registry 与 4.0 后相邻兼容规则 |
| `docs/open-source-readiness.md` | `LICENSE`、`VERSION`、`go.mod`、`Taskfile.yml`、`scripts/stage-release.ps1`、release/reproducible/CI workflows | 原页是上一轮遗漏，现已整体重写；区分 source-available、仓库可证明的 candidate/SBOM/attestation 与 GitHub/证书/真机等外部事实，并明确 workflow 不发布 GitHub Release |
| `docs/platform-support.md` | `.github/workflows/ci.yml`；build tags；`build/windows/*.manifest`；`internal/automation/target`、`automation/installed`；`internal/securestore`、`processsandbox` | 重写 host/Target 矩阵；使用 exact target/adapter literal，区分 Windows 11 支持政策、代码的 Windows-amd64 证据和 preview compile 边界 |
| `docs/product/workflows.md` | `internal/workflow/schema/{model,decode}.go`；`internal/workflow/authoring/patch.go`；`internal/application`；`internal/workflowbundle`；`internal/workflowstore` | 补 `derivedFrom`；修正 PreviewRun 不做 Target/credential/provider readiness，明确 compiler diagnostics 与 StartRun readiness 的边界；其余 Source/patch/bundle/recovery 主张核实 |
| `docs/product/targets-and-resources.md` | `internal/services/{settings,app}.go`；`internal/desktopapp/desktop.go`；`internal/appbootstrap/automation_runtime.go`；`internal/targetruntime`；`internal/automation/installed`；asset/macro/inputclip/snippet models | 改为 exact target/adapter/profile literal；修正 Settings 是“prepare → durable save → runtime publish”的有序提交，不是假装可回滚的跨层原子事务；区分 compiler 与 admission readiness |
| `docs/product/runs-and-schedules.md` | `internal/run/{record,journal,ledger}.go`；`internal/application/{application,run_admission,run_coordinator}.go`；`internal/workflow/compiler/debug.go`；`internal/services/schedule` | 区分 durable Run Record 与 in-process Target generation lease；修正 status category。Schedule 实际只顺序提交 start、不等 terminal，manual 经 FireNow，once 在每次注册（含 Reload）触发 |
| `docs/reference/cli.md` | `cmd/yotta/main.go`；`internal/localruntime/runtime.go`；`internal/storage/migrate` | 修正 Go FlagSet 的 flag-before-command 规则；明确 workflow commands 会 Ensure migration/打开 writer runtime，`validate` 只是不导入 Source，并非只读 profile |

## Cross-page findings

- 当前本地 `origin` 仍是 `git@github.com:Yuelioi/YHBox.git`，与 module/产品的 Yotta identity 不一致。本文档任务
  没有修改 remote、历史、tag 或发布状态；正式发布前必须由 owner 明确切换并验证目标仓库。
- Schedule `once` 的当前实现会在 daemon 初始注册和之后每次全量 `Reload` 时触发；任一 Schedule 保存导致的
  reload 可能再次启动 enabled once schedules。这是从 `daemon.go` 得出的当前行为，不把旧注释当事实。文档已
  如实描述，但是否改成“每进程只一次”仍是独立产品/实现决策；当前前端仍显示“启动后一次”，存在待决的
  产品文案/运行语义不一致。
- 根 `README.md` 和 `CONTEXT.md` 也按同一证据复核：移除不存在的通用 value expression 能力，修正 Schedule
  dispatch/once 以及“跨进程共用同一个 Application 实例”的表述；CI 中过期的 Workflow v3 step label 改为
  current contract，不把显示名称当 schema 事实。
- Settings runtime activation 在 durable settings commit 之后执行。activation 失败会返回 committed error，
  不是磁盘回滚；已同步修正 `PrepareInstallations` 的误导注释，避免未来再次写成跨层事务。
- GitHub repository settings、private vulnerability reporting、签名证书、维护者值守和真机 smoke 是本地仓库
  无法证明的外部状态；公开发布页只列验证方法，不冒充完成结论。

## Mechanical checks versus semantic evidence

`scripts/check-docs.mjs` 只负责链接、真实 Task 名和明确禁用旧引用；它不能证明上述语义。此次覆盖表和对应
生产调用链是语义审计证据，最终门禁结果记录在 Work handoff，而不是反过来用门禁证明文字正确。
