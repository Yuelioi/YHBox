# Compatibility and migration policy

Yotta 尚未发布稳定版。`internal/` Go API、Wails RPC 和节点实现接口可以在评审后演进，但已保存的用户数据不能静默损坏。

- Container graph 的 node `kind`、pin 名和声明 ID 属于持久化协议。变更必须提供迁移与旧 fixture 测试。
- Container package 使用 schema version 与 `yotta-lock.json` 校验完整 generation；混合或未知代进入 incompatible 状态，不猜测修复。
- Settings 使用快照、校验和原子替换。新增字段应保持旧 JSON 可读取并有明确默认值。
- 删除/重命名节点、错误码或 target kind 属于 breaking change，必须写 release note。
- 当前仓库实现中的 `yotta.workflow` v3、Catalog v3 与 `yotta.program` v3 是尚未发布的 Compiler 切片，不是需要长期兼容的产品格式。
- Yotta 3.1 stable 的目标协议代际统一为 Workflow Source 3.1、Catalog/Node Contract 3.1 与 Program 3.1，不建立另一套产品或节点版本号。
- 3.1 cutover 不读取或迁移 v3/legacy Container/Subgraph 表征，也不提供兼容 shim、dual-read/write 或 runtime fallback；旧 format/version 必须在 strict parse boundary 显式拒绝。切换纵向切片必须同时更新 parser、Compiler、Program opener、fixtures、breaking-change 文档和旧格式拒绝测试，不能在同一主线保留两个可执行事实。
- 当前扩展模型是 in-tree 编译；不保证第三方 Go plugin ABI。

稳定版发布前应定义 SemVer 边界、数据迁移支持窗口和废弃周期。
