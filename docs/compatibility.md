# Compatibility and migration policy

Yotta 尚未发布稳定版。`internal/` Go API、Wails RPC 和节点实现接口可以在评审后演进，但已保存的用户数据不能静默损坏。

- Container graph 的 node `kind`、pin 名和声明 ID 属于持久化协议。变更必须提供迁移与旧 fixture 测试。
- Container package 使用 schema version 与 `yotta-lock.json` 校验完整 generation；混合或未知代进入 incompatible 状态，不猜测修复。
- Settings 使用快照、校验和原子替换。新增字段应保持旧 JSON 可读取并有明确默认值。
- 删除/重命名节点、错误码或 target kind 属于 breaking change，必须写 release note。
- 新的 `yotta.workflow` v3、Catalog v2 与 `yotta.program` v3 是 3.0 epoch，不读取 legacy Container/Subgraph 表征，也不提供兼容 shim 或 migration；旧 format/version 必须显式拒绝。v3 子图调用的持久协议是 `core.call-subgraph` + `config.graphId` + stable GraphPort ID，旧 `Subgraph`、`CollapsedNode`、`Params` 和 marker normalization 不属于新协议。动态端口由 Catalog 中可哈希的 descriptor 声明，Program 冻结 Compiler 解析后的端口计划；当前只支持 Switch 的有序动态 Exec outputs。
- 当前扩展模型是 in-tree 编译；不保证第三方 Go plugin ABI。

稳定版发布前应定义 SemVer 边界、数据迁移支持窗口和废弃周期。
