# M5c — Rollback lineage 与计划重新授权

## Journey

Installation 切换 Release 后保留 immediate previous Release。用户可显式预览并确认 rollback；rollback
复用 staged update 的 diff、配置迁移、Readiness 和原子 CAS，不依赖 Registry 或网络。任何 Release
切换都使 exact run/schedule consent 失效，并持久暂停引用该 Installation 的已启用计划。

## Boundary

- Content Catalog schema 7 为 Installation 增加 nullable `previous_release_id`，引用已缓存 immutable Release；
  新安装为空，每次成功切换原子写入被替换的 Release。
- `PrepareRollback` 只能读取 Installation 的 previous Release，复用 `PrepareUpdate`；调用方不能指定任意
  digest 或绕过 migration conflict/readiness。
- appbootstrap 在 Release 已原子切换后调用 schedule pause port。schedule service 只把引用目标的 enabled
  schedule 改为 disabled，再统一 reload daemon；持久化已成功但 reload 失败使用 post-commit error。
- rollback 或 update 后 exact run/schedule consent 均为空；重新启用计划仍经过现有 schedule readiness
  authority 和显式 schedule consent。

## Verification

- schema 6 数据库无损迁移到 7；new install、update、rollback 的 current/previous lineage 正确并可离线读取。
- stale/faulted switch 不改变 current/previous；下架或网络不可用不影响已缓存 previous Release rollback。
- 多目标 schedule 只要引用更新的 Installation 就被暂停；无关或已 disabled schedule 不被改写。
- Catalog、深 Module、schedule composition/race、`task check` 与受影响真实旅程通过。

## Status

Finished.

## Evidence

- Content schema 7 migration fault/resume、Catalog current/previous lineage、stale transaction rollback 与
  deep Module update→rollback round trip 均由测试覆盖；rollback 不读取 Registry 或网络。
- Schedule 测试锁定只暂停引用目标 Installation 的 enabled 计划，无关/已暂停计划不变且 daemon 只 reload
  一次；desktop composition 在 Release commit 后调用未注册 Wails RPC 的专用 pauser。
- `go test -race ./internal/workflowinstallation ./internal/storage/catalog ./internal/services/schedule
  ./internal/appbootstrap` 与 `task check` 均退出 0；Wails contract 保持 17 services/156 methods/229 models，
  35 个受影响 Go 包通过。
