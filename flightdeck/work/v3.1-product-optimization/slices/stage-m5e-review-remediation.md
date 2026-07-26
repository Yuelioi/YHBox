# M5e — Review remediation

## Journey

用户从已安装工作流列表选择本机已缓存且身份匹配的候选 Release，先查看 graph/resource/variable、
dependency、target、credential 与 capability scope 的确定性差异和候选 readiness，再显式确认切换。
存在 immediate previous Release 时，同一入口可预览并确认 rollback。Release 切换后所有关联计划必须
全部持久暂停或全部保持原状；进程中断后重开可确定性完成已提交的批处理。

## Verification

- Wails/UI 暴露本机候选 Release 列表、update preview/apply 与 rollback preview/apply，apply 锁定预览时
  的 current/candidate Release identity，不接受 UI 伪造 Release bytes。
- Update diff 展示新增/移除 dependency、target definition、credential slot 与 exact capability
  ID/operations/target slot/credential slot/scope，并保留候选 readiness、迁移 conflict。
- Schedule Store 在 commit point 前失败不修改任何 schedule；commit point 后中断可在 reopen 时完成
  全部暂停；成功批处理只 reload daemon 一次。
- Catalog `Commit` 与 `SwitchRelease` 共享 tx-scoped immutable Release ensure，identity collision
  语义不变。
- 定向测试、`task check` 与按影响范围触发的 smoke 通过。

## Status

Finished.

## Evidence

- `WorkflowInstallationRepository.CacheRelease` 与 `Module.CacheVerifiedRelease` 只缓存已验证 immutable
  Release；Wails 只暴露候选摘要、一次性 preview token 和 apply，不接受候选 artifact bytes。
- `TestStagedUpdateListsCachedReleaseAndDiffsExactAuthorityBeforeSingleUseApply` 锁定 dependency 与
  capability scope added/removed、单次 token 和 rollback；appbootstrap 测试锁定动态 target slot、
  capability ID/operation/scope/risk/consent 的 exact projection。
- Schedule fault tests 锁定 precommit 零修改、postcommit interruption/reopen 恢复、恢复后继续保存不被
  journal 覆盖，以及一次 daemon reload；三个相关包 race 退出 0。
- `task check` 退出 0：38 个受影响 Go 包、前端 84 文件/358 项、Wails 17 服务/160 方法/236 模型通过。
- Windows WebView smoke `20260726-134114` 真实执行 cached 2.0 preview → apply → previous 1.0 rollback
  preview，退出 0；`workflow-installation-update.png` 与 `workflow-installation-rollback.png` 已目检。
- `task smoke:storage-migration` 在 review 阶段退出 0：dry-run、kill recovery、quarantine/resume、
  database health 与 migrated GUI 均通过。
