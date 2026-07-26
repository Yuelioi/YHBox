# 版本域解耦与自动化维护

## Goal

把产品发布版本、持久化合同、宿主接口、进程协议和存储布局拆成独立演进的版本域，并用生成脚本与
`task check` 门禁消除人工同步和裸版本字面量。

## Status

Finished

## Current

产品版本、artifact contract、Host Interface、wire protocol 与 store layout 已拆为独立 owner。根
`VERSION` 是唯一产品版本源；生成器、inventory、静态门禁、Windows 二进制资源检查和隔离桌面启动 smoke
已接入 Task。未发布的旧开发合同已切换到各域 v1，不提供迁移或 fallback。

## Next

无；后续产品发版使用 `task version:bump BUMP=<...>`，合同变化由所属 module 独立提升。

## Progress

- 2026-07-24 完成全仓 `3.1` 分布和传播路径审计：生产 Go 代码 90 行/49 文件，前端源码
  44 行/30 文件，另有生成合同、构建元数据和大量文档引用。
- 2026-07-24 整理并提交上一阶段四组改动：Windows 启动与应用授权、Workflow Source migration、
  长门禁等待规范、已完成 Flightdeck Work 清理。
- 2026-07-24 完成一手资料调研并接受 ADR 0006：产品 SemVer 不再兼任合同、协议或布局版本。
- 2026-07-24 建立根 `VERSION`、`yotta-versions` show/sync/bump/check/inventory 与 Task 自动化；
  Windows build 自动校验 fixed/string version resource、GUI subsystem 和隔离启动不秒退。
- 2026-07-24 将 Workflow、Data、Node、Catalog、Program、Capability、Run、Schedule、Host API、
  Script Worker、MCP document 和各 Store 切换为独立 v1；前端改用 generated `current` alias。
- 2026-07-24 `task build`、`task smoke:desktop` 和最终 `task check` 通过；最终门禁执行单元 209
  用时 238.7 秒、退出码 0。

## References

- [主流版本维护实践](references/versioning-mainstream-practices.md) — 产品版本、合同版本、协议协商和构建注入的一手资料。
- [兼容性约定](../../../docs/compatibility.md) — 独立版本域与稳定发布后的 migration 规则。
- [版本升级知识](../../knowledge/build/version-bump.md) — 产品版本自动维护入口。
