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
- 图严格区分 Data、Exec 与 Error；Status 是 Run event 而不是连线。pure-data 节点没有 Exec，任何层不得猜测或补端口。
- Execution Class、determinism 与 Capability Requirement 是独立维度。
- 每次工作先读 [领域语言](../../../CONTEXT.md)、[ADR-0001](../../adr/0001-compile-source-into-content-addressed-programs.md) 与[主流实践研究](../../research/node-system-mainstream-practices.md)。
- 实现与交付遵守仓库根 `AGENTS.md`，完整门禁只有 `task check`。
- 实施波次与发布门统一维护在 [Yotta 3.1 破坏性升级实施方案](../../../flightdeck/work/major-upgrade-review/plan.md)；本 map 只保存尚待解决的节点系统决策及其依赖。

## Decisions so far

<!-- closed tickets only; one linked gist per resolution -->

- [定义 Data Type 3.1 与 Value Envelope](tickets/define-data-types-and-value-envelope.md) — 采用带摘要的名义 TypeRef、JSON Schema TypeDefinition、独立 BindingState，以及 inline/blob/stream/handle 四分支 ValueEnvelope；禁止 `any` 降级和隐式转换。
- [定义 Node Contract 3.1 元模式](tickets/define-node-contract-metaschema.md) — NodeRef 固定 canonical semantic contract；端口通道显式分离，pure-data 不含 exec，安装实现锁由 Catalog/Program 持有，authoring/docs 不污染语义摘要。
- [定义 Capability 与 Target Planning](tickets/define-capability-and-target-planning.md) — Compiler 冻结 attributed least-privilege plan；Run policy 另行签发绑定 Program/plan/principal/provider/target/operation/scope 的短期 grant，所有 host 均无 ambient ServiceBundle 权限。
- [定义 Program 3.1 与 Run 语义](tickets/define-program-and-run-semantics.md) — 不可变 Program 经统一 admission 进入 generational RunRecord；Run Store CAS 持久化，Run Owner 独占临时 authority，遗留 RUNNING 只转 INTERRUPTED、不自动重放 effect。
- [原型验证 Schema 驱动的节点编辑体验](tickets/prototype-schema-driven-authoring.md) — 采用绑定精确 Catalog 的可严格重开 Authoring Projection；UI/文档共享端口、控件、默认提示、生命周期和 capability 事实，复杂交互只由不拥有语义的内置 Editor Adapter 承担。

## Fog

- 内置 137 个节点如何按新 contract 分批迁移，取决于类型、实例 contract 与 Program lowering 的最终形状。
- 新 Source schema、画布存储格式和测试 fixture 按已封存 Authoring Projection 逐批替换，不保留旧 presentation/pin registry 兼容层。
- debugger、局部重跑、血缘保留周期和运行历史 UI 的范围，取决于 Program 与 Run Value 设计。
