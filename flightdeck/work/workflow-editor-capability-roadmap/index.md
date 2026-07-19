---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。Slice 44 已实现管理页壳层和计划编辑可靠性修复：工作流、资源库、计划统一使用计划页的 `workspace-page__*` 标题层级；全局壳层不再挂运行日志，日志只在编辑器运行工作台按需加载；计划创建/编辑不再让 Vue Proxy 直接跨越 `structuredClone`。完整 `task check`（52 个前端测试文件、209 项测试）与 `task build` 通过，等待用户用最新 production build 真机接受。Slice 43 的成功反馈退出、刷新去重和 AdaptiveSelect 宽度修复仍一并等待复查；Slice 39/41 仍等待提权真机接受，Slice 37 保持发布阻断。

## Next

使用最新 UAC production build 和真实 workspace 接受 Slice 44：确认工作流、资源库、计划标题风格一致，三个管理首页没有全局日志，计划新建与编辑既有计划不再黑屏；同时复查 Slice 43 的成功状态条退出、刷新去重和普通枚举最长选项显示。

## Read now

- work/workflow-editor-capability-roadmap/slices/44-management-shell-and-schedule-reliability.md
- work/workflow-editor-capability-roadmap/slices/43-workflow-and-asset-library-management.md
- work/workflow-editor-capability-roadmap/context/library-management-stage.md
- knowledge/frontend/ui.md
- knowledge/frontend/headless-ui-verify.md
- knowledge/build/build.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询/调整完整 Slice registry
- work/workflow-editor-capability-roadmap/slices/39-recording-authoring-closure.md — 修改录制、模板和资源引用
- work/workflow-editor-capability-roadmap/slices/41-runtime-workbench-debug.md — 修改 Debug 工作台
- work/workflow-editor-capability-roadmap/slices/40-effective-target-inheritance.md — 修改 target Source/Compiler/Inspector
- work/workflow-editor-capability-roadmap/slices/42-stable-workspace-root.md — 修改 workspace 根目录与迁移
- work/workflow-editor-capability-roadmap/slices/27-architecture-recovery.md — 查看 R0–R5 恢复设计和架构理由
- work/workflow-editor-capability-roadmap/context/r0-worktree-ownership.md — 修改既有 dirty 路径前确认归属
- knowledge/architecture/installed-input-authority.md — 修改自动化目标授权/运行时边界
- knowledge/git/commits.md — 形成阶段或最终提交

## Progress

- Workflow Source、authoring tagged patch、Application 与 Workflow service 已耐久支持 description/category/tags；旧 Source 仍合法，更新继续走 CAS，列表额外投影节点数与 revision。
- Workflow Source 耐久保存 createdAt/updatedAt；创建、普通 patch、prepared patch、复制导入与替换导入由 Application 统一盖时间戳。旧 Source 不用文件时间伪造历史创建时间，修改后仅获得真实 updatedAt。
- WorkflowsView 与 AssetsView 已重构为纯列表工作台和互斥资源上下文，提供多维 facet、排序、可选列、批量动作与底部数字分页；未恢复 grid 与工作流快捷键。
- 工作流首页新增创建/修改日期列、最近 1/7/30/90 天筛选和按创建/修改时间排序；默认最近修改优先。
- 新增 AdaptiveSelect 共享组件并迁移 48 个普通 Select；真实 workspace 暴露的控件 chrome 漏算和工作流 fixed-width 覆盖已修正，普通筛选按最长选项加固定装饰余量并保留最大宽度。
- 两个列表共用上下文选择工具栏：选择后原位替换筛选行，清除选择归入选择状态，普通操作与危险删除分组；不再额外插入一条操作带。
- 资源与工作流批量元数据语义统一：分类支持保持/设置/清空，标签支持保持/添加/移除/替换/清空。资源不再用空值覆盖未修改字段；工作流经正式批量 RPC 逐项保留 name/description 并执行 revision/CAS 更新。
- 服务测试覆盖 1000 条工作流/资产规模；组件回归覆盖列表管理、选择态工具栏、成功反馈自动退出、刷新入口去重和自适应选择宽度。
- 工作流、资源库、计划统一使用 `workspace-page__*` 管理页标题契约；App 全局壳层不再挂 LogPanel，编辑器日志面板改为按需加载。
- 计划编辑使用 `shallowRef` 保存外来 DTO，并在 UI clone 边界解开 Vue Proxy；reactive Schedule 回归直接覆盖新建/编辑黑屏根因。
- 最新 `task check` 通过：52 个前端测试文件、209 项测试；bundle budget 与 `task build` 通过并生成最新 production Yotta.exe。
- 跳过 Launcher 子旅程后的 WebView 主旅程通过；完整 smoke 仍在 Launcher 执行新建空工作流时等待成功超时，该独立问题仍需发布前定位。

## Open questions

- Slice 43/44 与 Slice 39/41 仍需使用 UAC production build 和真实 workspace 真机接受；接受前不得声明 3.1 major upgrade 完成。
- 完整 WebView smoke 的 Launcher 子旅程为何无法观察到新建空工作流成功，需要按独立运行/启动器问题排查，不能通过永久跳过降低发布门禁。
- 旧 Workflow Source 的历史创建时间无法可靠恢复，UI 显示未知；不得改用文件时间回填。代码签名、公开仓库和 OSI 许可证替换仍属独立发布工程。
