---
title: 彻底升级 Yotta 节点系统
label: wayfinder:map
status: open
---

# 彻底升级 Yotta 节点系统

## Notes

- 目标是破坏性重建节点系统，不保留现有 Workflow Source、节点 ID、端口、配置或测试 fixture 的兼容层；仓库尚未发布，现有样本统一迁移。
- `Yotta 3.1`、`Node System 3.1`、Workflow/Node/Data/Program 3.1 使用同一版本名；它们替换尚未接入 runtime 的 v3 Compiler 切片，不建立双轨 runtime。
- Windows 是完整支持平台；Linux/macOS 保持平台中立核心可测试、GUI 可编译。
- 所有 Run 最终只能消费 Program Snapshot；不得保留旧 ContainerRunner 作为第二执行事实。
- Node Contract、Data Type、Authoring Projection、文档和第三方 Node Package 必须共享可序列化事实来源。
- 第三方执行同时覆盖最小 Wasm Node 与 Process Node；不做商店或在线分发。
- 插件不得注入任意前端 JavaScript；通用 UI 由 schema 生成，复杂交互只能使用 Yotta 内置 Editor Adapter。
- 图严格区分 Data、Exec、Error 与 Status；pure-data 节点没有 Exec，任何层不得猜测或补端口。
- Execution Class、determinism 与 Capability Requirement 是独立维度。
- 每次工作先读 [领域语言](../../../CONTEXT.md)、[ADR-0001](../../adr/0001-compile-source-into-content-addressed-programs.md) 与[主流实践研究](../../research/node-system-mainstream-practices.md)。
- 实现与交付遵守仓库根 `AGENTS.md`，完整门禁只有 `task check`。
- 实施波次与发布门统一维护在 [Yotta 3.1 破坏性升级实施方案](../../../flightdeck/work/major-upgrade-review/plan.md)；本 map 只保存尚待解决的节点系统决策及其依赖。

## Decisions so far

<!-- closed tickets only; one linked gist per resolution -->

- [定义 Data Type 3.1 与 Value Envelope](tickets/define-data-types-and-value-envelope.md) — 采用带摘要的名义 TypeRef、JSON Schema TypeDefinition、独立 BindingState，以及 inline/blob/stream/handle 四分支 ValueEnvelope；禁止 `any` 降级和隐式转换。

## Fog

- 内置 137 个节点如何按新 contract 分批迁移，取决于类型、实例 contract 与 Program lowering 的最终形状。
- 新 Source schema、画布存储格式和测试 fixture 的具体替换方式，取决于 Authoring Projection 原型。
- debugger、局部重跑、血缘保留周期和运行历史 UI 的范围，取决于 Program 与 Run Value 设计。
