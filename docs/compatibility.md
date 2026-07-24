# Compatibility and migration policy

Yotta 尚未发布稳定版。`internal/` Go API、Wails RPC、节点实现接口和未发布持久化格式都允许破坏性演进；
不再支持的数据必须在严格解析边界显式拒绝，不能静默猜测、修复或交给另一套 runtime。

## Independent version domains

- 产品发布版本是根目录 `VERSION` 中的 SemVer，只服务于构建、安装包和展示。
- Workflow Source、Data Type、Node Contract、Authoring Projection、Catalog、Program、Capability、
  Run Record 和 Schedule 分别拥有自己的 `format + version`；当前开发合同从各域 `1` 开始。
- Host Interface、Script Worker、Plugin 与 MCP document 由各自 module 拥有版本或协议身份。
- Blob、Workflow Source、Program 与 Run store 分别拥有 layout marker。
- Node Type 保留独立 SemVer 和 semantic digest；Compiler implementation 使用独立 build identity。
- inventory 可以同时展示所有版本域，但任何门禁都不得要求它们与产品版本相等。

## Migration rules

- 当前未发布的旧开发 artifact 不提供迁移、dual-read/write、fallback reader 或第二套 runtime。旧
  Container/Subgraph、历史 v3 compiler artifact 和曾使用产品版本作为合同版本的 workflow 都在 strict
  boundary 明确拒绝。
- Workflow Source Store 内置显式 `format + version` migration chain，但当前登记表为空。稳定发布后，
  shape 变化必须先提升所属合同版本，再登记确定性的相邻版本迁移；不得在相同合同身份下原地改变 schema。
- migration 只在内存副本上执行。完整链结果必须通过当前 schema、保持文件名绑定的 Workflow Source 身份，
  并由 Source Store 原子替换后才可进入活动集合。
- migration 失败、产出错误版本、产出无效当前合同或改变身份时，Store 启动失败并保留原文件；未登记版本作为
  unsupported artifact 隔离，不回退到旧 reader/runtime。
- 每条稳定版 migration 必须带固定旧版 fixture、链式升级测试、身份与 revision 保持断言，以及最终当前合同的
  canonical/strict validation。

## Other compatibility boundaries

- Settings 使用快照、校验和原子替换；新增字段应保持旧 JSON 可读取并有明确默认值。
- 删除或重命名节点、错误码、target kind 或 capability 语义属于 breaking change，必须写 release note。
- 内建节点采用 in-tree 显式装配；第三方扩展只接受已签名启用、锁定制品摘要的 Process/Wasm Node Package。
  不保证第三方 Go plugin ABI，也不接受插件 JavaScript/Vue/DOM 注入。
- Schedule target 只允许 `kind: "workflow"`。数字版本、`kind: "container"`、未知字段、文件名与实体 ID
  不一致均导致 Store 启动失败。
- GUI、headless CLI、MCP、AI authoring、schedule 与 hotkey 使用同一个 Application/Program Snapshot 命令面；
  旧 Container Run/Stop/Debug RPC 不存在。

稳定版发布前仍需定义每个对外版本域的支持窗口和废弃周期。
