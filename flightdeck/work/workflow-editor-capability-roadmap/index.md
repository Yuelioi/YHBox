---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。3.1 尚未发布；Stage 1–12 已完成 major upgrade、Windows 默认管理员权限收敛、权威类型系统、Type × Capability 闭包、Typed Authoring 与引用状态安全迁移。剩余 Projection 连接计划/Compiler 固定 fixture parity，以及 Slice 20 的管理员游戏窗口人工 smoke。

## Next

继续 Slice 25：建立同一份固定 fixture，让 TypeScript Projection 连接计划与 Go Compiler 对 direct/invalid 类型边界给出一致结论。阶段末批量验收并提交。

## Read now

- work/workflow-editor-capability-roadmap/slices/25-connection-plan-compiler-parity.md
- work/workflow-editor-capability-roadmap/artifacts/type-system-audit.md
- knowledge/nodes/typed-authoring-contract.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询 Slice 状态
- work/workflow-editor-capability-roadmap/slices/21-type-system-foundation.md — 修改类型关系、约束或 solver
- work/workflow-editor-capability-roadmap/slices/22-type-capability-closure.md — 修改结构字段、Break 节点或能力闭包门禁
- work/workflow-editor-capability-roadmap/slices/23-typed-authoring-ux.md — 修改连接计划、State 或转换交互
- work/workflow-editor-capability-roadmap/slices/24-settings-reference-integrity.md — 修改 application/target 删除语义
- work/workflow-editor-capability-roadmap/slices/20-desktop-target-uac-and-consent-ux.md — 进入 Windows 真实宿主 smoke
- knowledge/build/build.md — 进入阶段验收、发布、打包或真机 smoke

## Progress

- Stage 1–10 已恢复图编辑、调试、录制、资源、桌面/Android/Browser、多图和工作流库；Stage 11 固定 requireAdministrator manifest。
- 类型方向调研见 docs/research/visual-type-system-authoring.md；3.1 使用名义类型、显式关系、可见转换、可执行泛型和 typed Blackboard。
- Data Type semantic 已有 closed traits、assignable targets、StructureSpec；Catalog 拒绝悬空关系、未知 trait、结构字段漂移和能力缺口。
- Compiler 已执行 constraint、实例 scope、occurs check、唯一 LUB 和顺序无关绑定；Integer→Number 是显式安全提升，List 保持不变。
- Integer 运算、泛型 Log/Equal/ListContains/State、6 类结构化 Break 节点和 Type × Capability matrix 已闭环。
- EditorSession 做图级实例专化；Run 状态支持搜索、拖出 Read/Write、跨图引用定位与改型影响预览。
- 连接计划区分 direct/conversion/incompatible；有损/parser 显式选择后插入真实转换桥，整次一次 Undo。
- durable 精确输出支持 Promote to State：原子创建同类型状态、插入 State Write 并连线，一次 Undo。
- 引用状态改型会模拟目标类型并列出受影响数据边；需要转换或不兼容时 UI 阻止静默迁移。
- Application 对引用状态改型执行正式 Compiler 基线/候选诊断差分；只有无新增错误的原子候选可保存，PreparedPatch 也不能绕过。
- 状态列表和连接候选使用分段结果预算，搜索/模式变化会重置预算，避免一次渲染全部大集合。
- 最新完整 `task check` 于 2026-07-18 通过：Go 全量/vet/staticcheck、Workflow/Node contracts、AI eval、Wails bindings、前端测试、i18n、production build 与 bundle budget。

## Open questions

- Slice 20 的管理员游戏窗口 smoke 与最终发布前人工验收合并。
- 交互热路径解释 sealed Projection 以保持同步反馈；Compiler 是最终权威，Slice 25 用固定 fixture 防止跨语言漂移。
- 列表保持不变性；证明只读安全并在 contract 中表达后才考虑协变。
