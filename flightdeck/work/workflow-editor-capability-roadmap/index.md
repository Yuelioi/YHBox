---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。3.1 尚未发布；Stage 1–12 已完成 major upgrade、Windows 默认管理员权限收敛、权威类型系统、Type × Capability 闭包、Typed Authoring、引用状态安全迁移和 Projection/Compiler parity。代码阶段已闭环；唯一剩余项是 Slice 20 的管理员游戏窗口人工 smoke。

## Next

进入 Slice 20：使用管理员构建在真实管理员游戏窗口完成捕获、安装、输入与截图 smoke；通过后完成 Topic，失败则记录具体 adapter/UIPI 证据后修复。该步骤需要真实宿主，不能用单元测试替代。

## Read now

- work/workflow-editor-capability-roadmap/slices/20-desktop-target-uac-and-consent-ux.md
- knowledge/build/build.md
- knowledge/nodes/typed-authoring-contract.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询 Slice 状态
- work/workflow-editor-capability-roadmap/slices/21-type-system-foundation.md — 修改类型关系、约束或 solver
- work/workflow-editor-capability-roadmap/slices/22-type-capability-closure.md — 修改结构字段、Break 节点或能力闭包门禁
- work/workflow-editor-capability-roadmap/slices/23-typed-authoring-ux.md — 修改连接计划、State 或转换交互
- work/workflow-editor-capability-roadmap/slices/25-connection-plan-compiler-parity.md — 修改 TS/Go 直接连接边界
- work/workflow-editor-capability-roadmap/slices/24-settings-reference-integrity.md — 修改 application/target 删除语义
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
- TypeScript 与 Go 读取同一份 12-case 固定 fixture，覆盖 exact、nominal promotion、generic trait、union、List 不变性和 semantic digest。
- parity 阶段修复了 Compiler 对具体 List 的隐式协变；只有包含类型变量的 List 才递归绑定，具体 List 必须元素表达式完全相同。
- 状态列表和连接候选使用分段结果预算，搜索/模式变化会重置预算，避免一次渲染全部大集合。
- 最新完整 `task check` 于 2026-07-18 通过：162 frontend tests、Go 65.5% coverage/vet/staticcheck、Workflow/Node contracts、AI eval、Wails 167 models、i18n、production build 与 bundle budget。

## Open questions

- Slice 20 的管理员游戏窗口 capture/install/input/screenshot smoke 需要真实宿主完成。
- 列表保持不变性；证明只读安全并在 contract 中表达后才考虑协变。
