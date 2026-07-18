---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。3.1 尚未发布；Stage 1–11 已完成 major upgrade 和 Windows 默认管理员权限收敛。Stage 12 的权威类型系统基础批次已通过完整门禁：类型 semantic、关系图、trait/constraint、LUB 泛型求解、ConversionSpec、Integer 能力族、实例类型传播和 typed State Blackboard 已落地。结构化领域值字段消费和 Type × Capability 自动闭包门禁仍是发布前硬任务，不能把本 Topic 标记完成。

## Next

执行 Slice 22 剩余部分：把结构化类型字段变成 Catalog semantic 的显式类型契约，生成可执行 Break/field 节点与 Type × Capability matrix；禁止前端从 JSON Schema 或 TypeID 猜字段。随后完成 Slice 23 的 conversion plan/插入确认、引用计数与状态改型交互。阶段末再做一次批量验收。

## Read now

- work/workflow-editor-capability-roadmap/22-type-capability-closure.md
- work/workflow-editor-capability-roadmap/artifacts/type-system-audit.md
- knowledge/nodes/typed-authoring-contract.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询 Slice 状态
- work/workflow-editor-capability-roadmap/21-type-system-foundation.md — 修改类型关系、约束或 solver
- work/workflow-editor-capability-roadmap/23-typed-authoring-ux.md — 实现实例类型、拖线候选、转换插入或 Run 状态交互
- work/workflow-editor-capability-roadmap/24-settings-reference-integrity.md — 修改 application/target 删除语义
- work/workflow-editor-capability-roadmap/20-desktop-target-uac-and-consent-ux.md — 进入 Windows 真实宿主 smoke
- knowledge/build/build.md — 进入阶段验收、发布、打包或真机 smoke

## Progress

- Stage 1–10 已恢复图编辑、调试、录制、资源、桌面/Android/Browser、多图和工作流库；Stage 11 固定 requireAdministrator manifest。
- 调研 Unreal、Unity、LabVIEW、TypeScript、GraphQL、JSON Schema 与 Go 后冻结“名义类型、显式关系、可见转换、可执行泛型、typed Blackboard”方向；见 docs/research/visual-type-system-authoring.md。
- Data Type semantic 已加入 closed traits 和 assignable targets；Catalog 构造权威 Type System，拒绝悬空 target、关系环和未知 trait。
- Compiler 已执行 constraint、保持节点实例 scope、做 occurs check、唯一 LUB 和顺序无关重复绑定；Integer→Number 是明确安全提升，领域类型不按底层 primitive 偷偷兼容，List 保持不变。
- Node Contract 已加入 ConversionSpec（端口、lossless/lossy/parser、total、cost、autoInsert），seal 对照执行类别、错误和真实端口。
- Integer 新增保持整数类型的 add/subtract/multiply/modulo/negate/absolute/minimum/maximum/clamp；溢出按 JSON safe integer 失败。String/Number→Integer 策略节点显式且可失败。
- Log<T: Observable>、Equal<T: Equatable>、ListContains<T: Equatable>、State<T: Durable> 已执行真实约束；State declaration 拒绝非 durable 类型。
- EditorSession 已做图级固定点实例专化；State Read/Write 从 slot 得精确类型，连接消费者不会反向改写声明。拖线候选区分精确、泛型推导、安全兼容及转换风险。
- Run 状态支持搜索、拖出 Read、Alt 拖出 Write和显式插入操作；添加按钮使用 primary token。
- 删除 desktop application 会原子清理引用它的 automation targets，不再留下 settings 悬空引用。
- 2026-07-18 完整 task check 通过：Go 全量/覆盖率/vet/staticcheck、契约、AI eval、Wails bindings、i18n、155 frontend tests、production build 与 bundle budget 全绿。

## Open questions

- Slice 20 的管理员游戏窗口 smoke 与 Stage 12 发布前人工验收合并。
- 结构化字段契约必须显式表达 TypeExpression；JSON Schema 只负责值校验，不能充当跨领域类型推断器。
- 列表保持不变性；证明只读安全并在 contract 中表达后才考虑协变。
