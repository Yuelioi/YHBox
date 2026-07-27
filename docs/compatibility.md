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

## V4 direct-upgrade ledger

V4 当前允许从产品版本 `3.1.0` 直接升级。这个产品版本只描述升级支持窗口，不替代各 artifact 自己的
版本身份。兼容 reader 只有在下列条件全部满足后才能删除：

1. 发布策略把最低可直接升级版本提高到最后一个旧格式 producer 之后；
2. 至少一个仍在支持窗口内的中间版本已经发布并执行确定性的 durable rewrite；
3. 固定旧 fixture 证明 rewrite 后重开不再读取旧 authority；
4. recovery/backup/journal 中仍可能存在的旧 authority 已迁移，或明确随旧升级窗口一起停止支持；
5. 变更 release note 把旧 fixture 从“成功迁移”改为“明确 unsupported”，不能静默回退。

| Boundary | V4 接受的旧 authority | 成功后的 durable state | Reader 退役状态 |
| --- | --- | --- | --- |
| Settings | `yotta.settings/1` 中四类 installation 的 retired `workflowConsent`，producer `3.1.0` | 首次成功打开即经同一原子 Save 路径写成下一 generation；旧字段不进入当前 payload | 已有 rewrite 证据；提高 direct-upgrade floor 前保留 |
| Node Package | registry `2`，producer `3.1.0` | 完整验证 package generations、signature/trust 后原子发布 registry `3`，并固化 package-scoped grants | 已有 rewrite 证据；提高 direct-upgrade floor 前保留 |
| Run Store | root layout `1` 的 one-JSON-per-Run store | layout `1→2` 显式迁移导入当前 SQLite Run Ledger，并最后写 import marker；正常 Store 不读旧 JSON | 与 root layout `1` 支持一起退役 |
| Blob Store | root layout `2` 中 Blob layout `1` | layout `2→3` 显式迁移验摘要、移动到 sharded layout `2`、对账 inventory，最后发布 Blob marker `2` | 与 root layout `2` 支持一起退役 |
| Migration journal | document `1` 的 layout `1→2` recovery journal | 所有后续 journal state write 强制写 document `2` | 尚未完全可退役：不再变化的历史 v1 journal 仍需只读展示；删除前需在 writer lease 下迁移这些历史文件 |

## Other compatibility boundaries

- Settings 使用快照、校验和原子替换；新增字段应保持旧 JSON 可读取并有明确默认值。retired 字段的
  compatibility reader 必须返回 rewrite signal，不能只在内存中丢弃后继续依赖旧磁盘格式。
- 删除或重命名节点、错误码、target kind 或 capability 语义属于 breaking change，必须写 release note。
- 内建节点采用 in-tree 显式装配；第三方扩展只接受已签名启用、锁定制品摘要的 Process/Wasm Node Package。
  不保证第三方 Go plugin ABI，也不接受插件 JavaScript/Vue/DOM 注入。
- Schedule target 只允许 `kind: "workflow"`。数字版本、`kind: "container"`、未知字段、文件名与实体 ID
  不一致均导致 Store 启动失败。
- GUI、headless CLI、MCP、AI authoring、schedule 与 hotkey 使用同一个 Application/Program Snapshot 命令面；
  旧 Container Run/Stop/Debug RPC 不存在。

稳定版发布前仍需定义每个对外版本域的支持窗口和废弃周期。
