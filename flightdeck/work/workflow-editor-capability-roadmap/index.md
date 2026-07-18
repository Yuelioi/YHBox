---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。Slice 43 的耐久元数据契约、工作流首页与资源库列表工作台已经实现，阶段自动门禁和不含 Launcher 子旅程的 WebView smoke 通过；等待用户在 UAC production build 和真实 workspace 上接受。Slice 39/41 同样仍等待提权真机接受，Slice 37 继续保持发布阻断。完整 WebView smoke 另有悬浮启动器运行新建空工作流超时，已作为独立发布前问题保留，不能用本阶段通过掩盖。

## Next

使用 UAC production build 和真实 workspace 验收 Slice 43：创建/编辑工作流元数据，检查分类/标签筛选、列控制和数字分页；切换录制/视觉模板资源上下文并检查列表、筛选、分页和批量选择。接受后再关闭 Slice 43，并继续解决完整 WebView smoke 的 Launcher 超时及 Slice 39/41 真机项。

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
- WorkflowsView 已重构为纯列表工作台：单一创建主动作、创建/编辑 Modal、导入 overflow、全文搜索、分类/多标签 facet、排序、可选列、批量动作和底部数字分页；未恢复 grid 与工作流快捷键。
- AssetsView 已重构为录制/视觉模板互斥上下文和高密度列表，完整展示名称、描述、分类、标签与类型信息，并提供 facet、排序、批量动作和 20/50/100 底部分页。
- 服务测试覆盖 1000 条工作流/资产规模；真实 Nuxt UI 组件测试覆盖创建 Modal、元数据显示、上下文隔离与第 2 页查询。task check 通过：47 个前端测试文件、195 项测试全部通过。
- task build 通过并生成 production Yotta.exe；WebView 主旅程在显式跳过 Launcher 子旅程后通过，检查了工作流恢复页、编辑器与资源库截图。
- 完整 WebView smoke 当前在悬浮启动器执行仅含 Run 开始节点的新建工作流时等待成功超时；首页与资源库路径未报错，但该独立问题仍需发布前定位。

## Open questions

- Slice 43 与 Slice 39/41 仍需使用 UAC production build 和真实 workspace 真机接受；接受前不得声明 3.1 major upgrade 完成。
- 完整 WebView smoke 的 Launcher 子旅程为何无法观察到新建空工作流成功，需要按独立运行/启动器问题排查，不能通过永久跳过降低发布门禁。
- 工作流本阶段不伪造无法从现有 Source 可靠恢复的历史创建日期；代码签名、公开仓库和 OSI 许可证替换仍属独立发布工程。
