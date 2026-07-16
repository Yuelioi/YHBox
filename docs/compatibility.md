# Compatibility and migration policy

Yotta 尚未发布稳定版。`internal/` Go API、Wails RPC、节点实现接口和未发布持久化格式都允许破坏性演进；任何不再支持的数据必须在严格解析边界显式拒绝，不能静默猜测、修复或交给另一套 runtime。

- legacy Container graph/package 不属于 3.1 可读格式，也没有迁移承诺；相关 editor、store、RPC 与 runtime 已整体删除。
- Settings 使用快照、校验和原子替换。新增字段应保持旧 JSON 可读取并有明确默认值。
- 删除/重命名节点、错误码或 target kind 属于 breaking change，必须写 release note。
- 当前 durable contract 为 Workflow/Data/Node/Catalog/Program 3.1；此前未发布的 v3 Compiler artifact 已破坏性删除，不提供兼容读取或双 runtime。
- Data Type 3.1 machine artifact 必须显式携带 `schemaRoot`；依赖 schema bundle 排序推断根的早期 3.1 artifact 已破坏性拒绝，不做 dual-read 或 root fallback。
- Yotta 3.1 stable 的目标协议代际统一为 Workflow Source 3.1、Catalog/Node Contract 3.1 与 Program 3.1，不建立另一套产品或节点版本号。
- 3.1 cutover 不读取或迁移 v3/legacy Container/Subgraph 表征，也不提供兼容 shim、dual-read/write 或 runtime fallback；旧 format/version 必须在 strict parse boundary 显式拒绝。切换纵向切片必须同时更新 parser、Compiler、Program opener、fixtures、breaking-change 文档和旧格式拒绝测试，不能在同一主线保留两个可执行事实。
- 内建节点采用 in-tree 显式装配；第三方扩展只接受已签名启用、锁定制品摘要的 Process/Wasm Node Package。不保证第三方 Go plugin ABI，也不接受插件 JavaScript/Vue/DOM 注入。
- Schedule durable contract 同步升级为 `schemaVersion: "3.1"`，target 只允许 `kind: "workflow"`。数字版本、`kind: "container"`、未知字段、文件名与实体 ID 不一致均导致 Store 启动失败；不提供 dual-read 或自动迁移。
- `8cdff7e2` 起生产 composition 不再装配 legacy `ContainerRunner`、旧 debug manager 或旧 MCP HTTP server。GUI、headless CLI、MCP、AI authoring、schedule 与 hotkey 均使用同一个 Application/Program Snapshot 命令面；旧 Container Run/Stop/Debug RPC 不存在。

稳定版发布前应定义 SemVer 边界、数据迁移支持窗口和废弃周期。
