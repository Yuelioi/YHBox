# 3.1 产品能力连续性审计与升级路线

## Goal

审计 3.0 能力与 3.1 现状，按唯一新架构、用户价值和安全边界决定恢复、重做或删除。

## Status

Finished

## Current

R0–R5 与历史交付 1–44 全部完成。Capability ledger 的 P0/P1 项均已 verified、明确替代或明确
删除；最终 WebView、Windows native、UAC production、`task check` 和 production build 证据完整。
本 Work 只作为 3.0→3.1 能力恢复历史，不承载新的体验扩张。

## Next

None.

## Progress

- 固定 3.0 oracle、capability ledger、golden journeys 和 dirty-worktree ownership。
- 重建 Typed RPC、Installation、Recording Session、Asset Picker 与 Windows native 闭环。
- 完成编辑器、Android/ADB、Browser CDP、target inheritance 和运行工作台能力恢复。
- 建立 Workflow/Asset 列表管理、批量元数据、稳定 workspace root 和计划引用。
- 完整 WebView 旅程闭合 Debug、Launcher、资源工作区、保存重开和计划引用。
- 最终 `task check`、production build、Windows native smoke 与 UAC production 启动通过。
- 历史能力被明确恢复、替代或删除，不再通过模糊延期保持开放。

## References

- [Architecture health audit](architecture-health-audit.md) — 最终架构事实基线。
- [Capability audit](capability-audit.md) — 历史能力差距。
- [Research](research.md) — 产品与技术依据。
- [Capability ledger](references/capability-ledger.md) — 恢复、替代和删除结果。
- [Golden journeys](references/golden-journeys.md) — G01–G17 验收路径。
- [Architecture recovery](references/27-architecture-recovery.md) — R0–R5 重建设计。
- [Final management shell](references/44-management-shell-and-schedule-reliability.md) — 最后交付边界。
