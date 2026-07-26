# R8 — Blob layout 发布迁移修复

## Goal

修复 production EXE 在已提交 root layout 2、但 Blob Store 仍持有 v1 ownership marker 时启动失败；
通过正式 root-layout migration 保存已有对象、恢复语义与版本权威，不在 `blob.Open` 中暗改用户数据。

## Status

Finished

## Root cause

R4 将 Blob Store 从 v1 根目录平铺对象升级为 v2 两级分片、staging/trash 与 Catalog inventory，同时把
marker 从 `yotta/blob-store/1` 提升为 `/2`。R7 的 root layout 1→2 migration 提交了 root manifest 和
Catalog/Run 数据迁移，却没有迁移 Blob 物理布局或 marker。于是 root health 已接受 layout 2，后续
`blob.Open` 又因精确 marker 比较拒绝启动。

## Decisions

- root layout 提升到 3，保留既有 immutable 1→2 step，再注册独立 2→3 step；1 跳到当前版本时按两个
  reviewed step 连续执行。
- Blob 模块提供显式、仅供 migration coordinator 调用的 v1→v2 物理迁移；`Open` 继续严格只接受当前布局。
- v1 平铺对象在移动前验证 basename、regular-file authority、size 与 SHA-256；已分片对象同样复验并对账
  durable inventory。marker 最后发布，根 manifest 在完整 verify 后最后发布。
- 迁移可重入；提交前回滚把对象恢复为平铺权威并由 snapshot 恢复 Catalog。旧 document v1 journal
  保持可读，新 journal 使用 version 2 记录 `blobLayoutFrom`。

## Evidence

- 精确最小复现先稳定跑红：v1 marker 进入真实 `blob.Open` 得到
  `blob store directory has an unsupported ownership marker`。
- Blob 单测覆盖空库 marker、带对象迁移/读取/回滚、已分片 inventory 对账。
- Root migration 测试覆盖真实 layout 2 + v1 flat object、三个 durability kill point、resume、
  rollback/reapply、旧 1→2 journal 和 1→2→3 连续路由。
- `go test -race ./internal/blob ./internal/storage ./internal/storage/migrate -count=1` 退出 0。
- `task check` 通过 56 个受影响 Go 包；`task build` 生成 production `bin/Yotta.exe`，版本与隔离启动通过。
- `task smoke:storage-migration` 完成 layout 1→2→3、recovery GUI kill、quarantine/resume、双库 health
  与迁移后 GUI 重启。
- 真实 `%LocalAppData%\Yotta\Yotta` 先以副本验证 plan/apply/health，再由 production EXE 提交
  layout 2→3；root manifest 为 3、Blob marker 为 2、journal committed，Content Catalog schema 7 与
  Run Ledger schema 2 healthy。迁移当次存活 8 秒，随后二次重开存活 6 秒。

## Next

返回 [Stage M6](../plan.md)；本切片无剩余动作。
