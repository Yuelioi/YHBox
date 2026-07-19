---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。Slice 43 的耐久元数据契约、工作流首页与资源库列表工作台已经实现；工作流与资源库现在共用上下文选择工具栏，并支持安全的批量分类/标签编辑。完整阶段门禁和 production build 通过，等待用户在 UAC production build 和真实 workspace 上接受。Slice 39/41 同样仍等待提权真机接受，Slice 37 继续保持发布阻断。完整 WebView smoke 另有悬浮启动器运行新建空工作流超时，已作为独立发布前问题保留，不能用本阶段通过掩盖。

## Next

使用 UAC production build 和真实 workspace 验收 Slice 43：检查工作流与资源库的跨页多选、选择态工具栏、批量分类的保持/设置/清空，以及标签的保持/添加/移除/替换/清空；同时复查筛选、列控制和数字分页。接受后再关闭 Slice 43，并继续解决完整 WebView smoke 的 Launcher 超时及 Slice 39/41 真机项。

## Read now

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
- WorkflowsView 与 AssetsView 已重构为纯列表工作台和互斥资源上下文，提供多维 facet、排序、可选列、批量动作与底部数字分页；未恢复 grid 与工作流快捷键。
- 两个列表共用上下文选择工具栏：选择后原位替换筛选行，清除选择归入选择状态，普通操作与危险删除分组；不再额外插入一条操作带。
- 资源与工作流批量元数据语义统一：分类支持保持/设置/清空，标签支持保持/添加/移除/替换/清空。资源不再用空值覆盖未修改字段；工作流经正式批量 RPC 逐项保留 name/description 并执行 revision/CAS 更新。
- 服务测试覆盖 1000 条工作流/资产规模；真实 Nuxt UI 组件测试覆盖列表管理和选择态工具栏。task check 通过：48 个前端测试文件、200 项测试；task build 生成最新 production Yotta.exe。
- 跳过 Launcher 子旅程后的 WebView 主旅程通过；完整 smoke 仍在 Launcher 执行新建空工作流时等待成功超时，该独立问题仍需发布前定位。

## Open questions

- Slice 43 与 Slice 39/41 仍需使用 UAC production build 和真实 workspace 真机接受；接受前不得声明 3.1 major upgrade 完成。
- 完整 WebView smoke 的 Launcher 子旅程为何无法观察到新建空工作流成功，需要按独立运行/启动器问题排查，不能通过永久跳过降低发布门禁。
- 工作流本阶段不伪造无法从现有 Source 可靠恢复的历史创建日期；代码签名、公开仓库和 OSI 许可证替换仍属独立发布工程。
