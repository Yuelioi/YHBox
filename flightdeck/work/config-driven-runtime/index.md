# 配置直驱运行时

## Goal

把网络、应用和自动化目标从 capability、授权、grant、consent、租约与 arm 体系中完整移除，使三类运行
能力只由当前设备配置和实际可用性决定，并修复由旧安全模型造成的 Run 中断与误导反馈。

## Status

Finished

## Current

网络、应用和自动化目标已经统一迁移为 Configured Target：Node Contract 声明 slot/kind/operation，
Run 从不可变配置快照直接打开、调用并关闭 provider。三类目标不再进入 Capability Plan、Admission、
Policy、Consent、Credential Binding、Run Grant、Resource Broker 或 TTL。

固定 5 分钟 GrantTTL、资源句柄过期、应用 SHA/身份检查、网络私网与 redirect 限制、Automation
identity pinning 和对应 UI/RPC 字段均已删除。缓存编辑器会在重新激活时刷新干净会话，保存 revision
冲突会基于最新 revision 自动重放同一语义 patch 一次。

## Next

无。实现入口见 [Configured Target runtime](../../../internal/targetruntime)、
[节点运行时](../../../internal/noderuntime) 和
[架构说明](../../../docs/architecture/README.md)。

## Progress

- 2026-07-30 从真实失败 Run 确认固定 5 分钟 GrantTTL 会在循环第 3 次点击模板时让资源句柄过期，
  随后被错误映射为 `automation.contract_violation`。
- 2026-07-30 用户将修复范围明确扩大为网络、应用和自动化目标三整套安全/授权体系，三者统一改为
  配置直驱，不接受把 TTL 延长、续租或隐藏授权作为替代方案。
- 2026-07-31 新增统一 `internal/targetruntime` 配置快照与 per-Run direct invocation；三类 Node
  Contract 改用 `configuredTargets`，Capability Catalog 只保留 AI、File、Blob 与 Stream 等非目标资源。
- 2026-07-31 网络恢复标准 HTTP client 行为并允许本机、私网、代理、redirect 和远程 CDP；应用只保存
  路径与 argv 并继承完整环境；Automation 不再固定设备/浏览器身份，也没有隐藏 3 秒 CDP 超时。
- 2026-07-31 删除 Grant/Handle 的过期语义和生产 5 分钟 TTL；历史 `expiresAt` 仅作为旧 JSON
  兼容字段读取，不参与执行或新记录。
- 2026-07-31 修复 KeepAlive 编辑器 revision 陈旧与保存冲突自动重放。Go `internal/...`、前端
  105 个测试文件/457 项测试以及仓库增量 `task check` 全部通过。
- 2026-07-31 修复升级后的启动恢复回归：旧应用配置的 `executableDigest` 和旧网络配置的
  `allowPrivateNetwork` 现在与 `workflowConsent` 一样，在校验原 checksum 后单向移除并发布下一
  settings generation。真实 generation 161 的 primary/backup 隔离副本成功打开，原用户文件未改动；
  服务、local runtime、desktop composition 与完整 `task check` 再次通过。
