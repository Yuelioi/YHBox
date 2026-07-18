---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

In progress。用户真机测试 1 的 Slices 39–42 已完成实现并通过阶段自动验收：录制/资产创作、workflow 默认目标、运行工作台/单步反馈与稳定 workspace 根已形成闭环。Topic 仍等待用户在当前 UAC build 上做最终真机接受，不能据此提前声明 3.1 发布完成。

## Next

使用已启动的提权版 Yotta 做最终真机复测：分别从资源库和编辑器录制、编辑并回放简易/精准 clip；显式确认模板；用一个 workflow default 驱动 Click/Keys/Template 并验证单节点 override；检查普通 Run、失败、连续 Step 和迁移后的工作流。用户接受后完成 Slice 37 的最终发布门禁与知识退役判定。

## Read now

- work/workflow-editor-capability-roadmap/slices/37-release-gate-knowledge-retirement.md
- work/workflow-editor-capability-roadmap/slices/38-user-device-test-1.md
- work/workflow-editor-capability-roadmap/slices/39-recording-authoring-closure.md
- work/workflow-editor-capability-roadmap/slices/40-effective-target-inheritance.md
- work/workflow-editor-capability-roadmap/slices/41-runtime-workbench-debug.md
- work/workflow-editor-capability-roadmap/slices/42-stable-workspace-root.md
- work/workflow-editor-capability-roadmap/context/user-device-test-1-research.md
- knowledge/build/build.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询/调整完整 Slice registry
- work/workflow-editor-capability-roadmap/slices/40-effective-target-inheritance.md — 修改 target Source/Compiler/Inspector
- work/workflow-editor-capability-roadmap/slices/41-runtime-workbench-debug.md — 修改 Run/Timeline/Debug/Logs
- work/workflow-editor-capability-roadmap/slices/42-stable-workspace-root.md — 修改 workspace 根目录与迁移
- work/workflow-editor-capability-roadmap/slices/27-architecture-recovery.md — 查看 R0–R5 恢复设计和架构理由
- work/workflow-editor-capability-roadmap/context/r0-worktree-ownership.md — 修改既有 dirty 路径前确认归属
- knowledge/frontend/monotonic-rpc-event-snapshots.md — 处理 debug RPC/event 竞态
- knowledge/architecture/installed-input-authority.md — 修改自动化目标授权/运行时边界
- knowledge/git/commits.md — 形成阶段或最终提交

## Progress

- R1–R4 的 Typed RPC、Installation Manifest、Target Runtime、Recording/Asset 基础设施、Windows/Android/Browser Adapter、typed authoring 与规模门禁已完成。
- 用户真机测试 1 的源码审计确认核心 Source → Compiler → Program、automation adapter、asset store 与 run journal 架构可继续使用，问题来自五个未收拢的浅边界。
- Slice 39 已将 completion 收到单一 RecordingSession owner；资源库只保存，编辑器可保存并插入；简易宏支持 key/click/scroll、增删重排、delay/duration，精准录制保留完整轨迹并折叠预览；模板 picker 有显式选择与确认。
- Slice 40 已加入版本化 targetDefaults、Compiler effective config 与 Inspector inherited/override/restore；录制节点默认继承 workflow target。
- Slice 41 已合并 Logs/Timeline/Debug 底部工作台；普通 Run 不抢焦点，失败开 Logs，pause 开 Debug；Timeline 分页有界；单步明确“刚执行/即将执行”。
- Slice 42 已把 canonical 数据根改为 workspace，并在真实 bin/data 上原子迁移 workspace-3.1；用户 workflow 仍存在。
- 阶段门禁已通过 task check、production build、WebView 编辑器 smoke、Windows native automation smoke 与人工截图检查。

## Open questions

- 用户对当前提权 build 的最终真机接受；接受后才能关闭 Slice 37 并重新判断 3.1 是否达到发布标准。
- 代码签名、公开仓库、OSI 许可证替换仍属独立发布工程，不在本 Topic 内擅自推进。
