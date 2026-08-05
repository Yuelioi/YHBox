# Compatibility and migration policy

Yotta 尚未发布稳定版。`internal/` Go API、Wails RPC、节点实现接口和未发布合同可以破坏性演进；被淘汰的
开发期 artifact 必须在 strict boundary 明确拒绝，不能静默猜测、修复或交给第二套 runtime。

## Independent version domains

- 产品版本只来自根 `VERSION`，用于构建、安装包和展示。
- Workflow Source、Data Type、Node Contract、Authoring Projection、Catalog、Program、Run Record、Schedule、
  settings、package registry、worker/plugin/MCP protocol 和 store layout 各自拥有 `format + version`。
- Node Type 使用稳定 URI、独立 SemVer 和 semantic digest；Compiler 使用独立 build identity。
- 这些版本域不与产品版本同步。当前值从所属 package/schema 读取，`task versions:inventory` 只做聚合展示。

## Change rules

- shape 或语义变化先提升所属合同版本，再添加确定性的相邻 migration；不能在同一 identity 下原地改 schema。
- migration 在副本或 staging authority 上执行，结果通过当前 strict validation、identity/revision 与引用完整性
  检查后才能 durable publish。
- migration 失败、版本未知、checksum 漂移、身份变化或未来 schema 时保留原事实并 fail closed/quarantine；
  不增加 fallback reader、dual-write 或旧 runtime。
- 稳定 migration 必须有冻结旧 fixture、preflight/dry-run、链式升级、kill-point resume/recovery 和重开测试。
- settings 等兼容 reader 如果接受 retired field，成功读取后必须 durable rewrite；不能只在内存丢字段。
- 删除/重命名节点、端口、错误码、Target kind、capability 或 durable field 是 breaking change，需要 release note
  和明确旧 artifact 行为。

## Current supported upgrade seam

当前 storage root 支持注册过的 layout 1 → 2 → 3 迁移；其中 Run JSON authority 导入 SQLite Run Ledger，
Blob 平铺 layout 迁到 `objects/sha256` 分片布局。settings 仍能读取并重写明确登记的 3.1 retired installation
字段。精确 registry、journal 和 schema 路由以 `internal/storage/migrate/`、`internal/services/settings_store.go`、
`internal/nodepackage/` 及其冻结 fixture 为权威。

兼容 reader 只有在最低直接升级版本已越过最后一个旧 producer、所有 backup/recovery authority 已迁移、
且 fixture 从“成功迁移”改为“明确 unsupported”后才能删除。稳定版发布前还需定义每个对外版本域的支持
窗口和废弃周期。
