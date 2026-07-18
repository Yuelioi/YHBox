---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。3.1 尚未发布；Stage 1–11 已完成 major upgrade 和 Windows 默认管理员权限收敛。Stage 12 的权威类型系统与 Type × Capability 闭包已经完成；Typed Authoring 已交付基于 sealed Projection 的连接计划、显式转换策略插入和 Promote to State。剩余状态改型影响预览与大目录预算/虚拟化仍需完成，不能把本 Topic 标记完成。

## Next

继续 Slice 23：为状态删除/改型提供图级影响预览和可撤销的安全重写；随后给大 Catalog/状态搜索加入结果预算、取消与虚拟化。交互热路径继续消费 sealed Authoring Projection，Compiler 保持保存/运行前最终权威；阶段末批量验收，不为每个小改动重复跑完整门禁。

## Read now

- work/workflow-editor-capability-roadmap/23-typed-authoring-ux.md
- work/workflow-editor-capability-roadmap/artifacts/type-system-audit.md
- knowledge/nodes/typed-authoring-contract.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询 Slice 状态
- work/workflow-editor-capability-roadmap/21-type-system-foundation.md — 修改类型关系、约束或 solver
- work/workflow-editor-capability-roadmap/22-type-capability-closure.md — 修改结构字段、Break 节点或能力闭包门禁
- work/workflow-editor-capability-roadmap/24-settings-reference-integrity.md — 修改 application/target 删除语义
- work/workflow-editor-capability-roadmap/20-desktop-target-uac-and-consent-ux.md — 进入 Windows 真实宿主 smoke
- knowledge/build/build.md — 进入阶段验收、发布、打包或真机 smoke

## Progress

- Stage 1–10 已恢复图编辑、调试、录制、资源、桌面/Android/Browser、多图和工作流库；Stage 11 固定 requireAdministrator manifest。
- 调研 Unreal、Unity、LabVIEW、TypeScript、GraphQL、JSON Schema 与 Go 后冻结“名义类型、显式关系、可见转换、可执行泛型、typed Blackboard”方向；见 docs/research/visual-type-system-authoring.md。
- Data Type semantic 已加入 closed traits、assignable targets 和显式 StructureSpec；字段使用稳定 port ID、JSON key 与 TypeExpression，Schema 只校验值而不负责推断领域类型。
- Catalog 构造权威 Type System，拒绝悬空 target、关系环、未知 trait、未知结构字段类型及对象 schema/结构契约漂移。
- Compiler 已执行 constraint、保持节点实例 scope、做 occurs check、唯一 LUB 和顺序无关重复绑定；Integer→Number 是明确安全提升，领域类型不按底层 primitive 偷偷兼容，List 保持不变。
- Node Contract 已加入 ConversionSpec；Integer 有保持整数类型的运算族，Log/Equal/ListContains/State 使用真实 trait constraint。
- Point、Region、TemplateMatch、QRCode、ColorBlob、FileMetadata 均有显式字段契约；Catalog 自动生成 6 个可执行 Break 节点并由 runtime 投影字段。
- Type × Capability 构建门禁覆盖 producer/consumer、literal、durable/observable/equatable、numeric/ordered、structure/break；新增公开类型缺能力或理由明确的 waiver 时 Catalog 构建失败。
- EditorSession 已做图级实例专化；State Read/Write 从 slot 得精确类型。Run 状态支持搜索、拖出 Read、Alt 拖出 Write、引用计数、定位首个引用，并阻止删除仍被引用的状态。
- 连接结果区分 direct/conversion/incompatible 并带稳定原因、源/目标类型和按风险/成本排序的 ConversionSpec 候选；有损/可失败策略不静默执行。
- 用户拖到可转换目标时选择策略，编辑器插入真实转换节点和两条边，整次操作一次 Undo；String→Number 可明确选择整数 parser 后安全提升或直接 number parser。
- durable 精确输出可从拖线候选中 Promote to State：原子创建同类型状态、插入 State Write 并连线，整次操作一次 Undo；runtime-only 或无合法默认值类型不会出现该操作。
- 删除 desktop application 会原子清理引用它的 automation targets，不再留下 settings 悬空引用。
- 2026-07-18 两个 Stage 12 阶段门禁均完整通过；最新一次包含 159 frontend tests、Go 全量/覆盖率/vet/staticcheck、契约、AI eval、Wails bindings、i18n、production build 与 bundle budget。

## Open questions

- Slice 20 的管理员游戏窗口 smoke 与 Stage 12 发布前人工验收合并。
- 交互热路径从 sealed Projection 解释关系、constraint 和 ConversionSpec，以保持同步反馈；Compiler 是最终权威，前端不得持有手写 TypeID 关系表。
- 自动插入仅限唯一、总函数、无损、确定、无副作用且无需 capability 的单步转换；当前所有有损/parser 都要求用户显式选择。
- 列表保持不变性；证明只读安全并在 contract 中表达后才考虑协变。
