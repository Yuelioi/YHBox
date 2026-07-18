---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。用户真机测试 2 的代码整改和阶段自动门禁已完成：Debug Start 的 RPC/event 竞态已修复；模板整项选择、资源引用字段、资源库高密度工作台和任务导向调试器已落地。Slice 39 与 Slice 41 保持 in_progress，等待用户使用当前 UAC production build 和真实 workspace 数据完成接受；Slice 37 继续保持发布阻断，接受前不得声明 3.1 major upgrade 完成。

## Next

启动当前 UAC production build，由用户用真实数据完成三个接受旅程：Debug Start 立即进入 paused 且 Step/Continue 可用；点击模板整卡或双击可以选择并绑定；资源库类型切换、密集列表/模板网格和 Inspector 复合资源字段符合日常管理。接受后将 Slice 39/41 标为 completed，再继续 Slice 37 的最终发布门禁与历史 Knowledge 退役。

## Read now

- work/workflow-editor-capability-roadmap/slices/37-release-gate-knowledge-retirement.md
- work/workflow-editor-capability-roadmap/slices/39-recording-authoring-closure.md
- work/workflow-editor-capability-roadmap/slices/41-runtime-workbench-debug.md
- work/workflow-editor-capability-roadmap/context/user-device-test-2-design.md
- knowledge/frontend/monotonic-rpc-event-snapshots.md
- knowledge/frontend/ui.md
- knowledge/frontend/headless-ui-verify.md
- knowledge/build/build.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询/调整完整 Slice registry
- work/workflow-editor-capability-roadmap/slices/40-effective-target-inheritance.md — 修改 target Source/Compiler/Inspector
- work/workflow-editor-capability-roadmap/slices/42-stable-workspace-root.md — 修改 workspace 根目录与迁移
- work/workflow-editor-capability-roadmap/slices/27-architecture-recovery.md — 查看 R0–R5 恢复设计和架构理由
- work/workflow-editor-capability-roadmap/context/r0-worktree-ownership.md — 修改既有 dirty 路径前确认归属
- knowledge/architecture/installed-input-authority.md — 修改自动化目标授权/运行时边界
- knowledge/git/commits.md — 形成阶段或最终提交

## Progress

- Debug Start 现在在 RPC 返回前缓存同 runID 的早到 snapshot，并按 generation 单调合并；精确回归测试覆盖 event-before-RPC。
- AssetPickerModal 支持模板/录制整项点击、双击和键盘选择，固定 footer 显示明确确认动作，多分辨率 variant 降为次级选择。
- Inspector 使用单一复合资源字段显示预览、名称、元数据、状态和更换/清除，不再重复展示资源引用。
- 资源库改为类型上下文工作台：Input Recording 与 Visual Template 互斥，默认 50 行高密度列表，带搜索、筛选、排序、分页、多选 batch bar 和行内菜单；模板可切换网格。
- Debug 面板按状态组织控制和执行位置；终态不再显示伪造的 previous/next 占位或“即将执行”，空快照不再平铺四列“无”。
- 阶段验证通过：前端类型检查和 lint；Vitest 46 files / 193 tests；task check；task build；production-like WebView workflow smoke；人工检查 Assets 与 Debug 终态截图。

## Open questions

- 当前阶段只差提权真机接受；接受前不得再次把 Slice 39/41 或 3.1 major upgrade 声明为完成。
- 代码签名、公开仓库、OSI 许可证替换仍属独立发布工程，不在本 Topic 内擅自推进。
