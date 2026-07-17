---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做、延期或删除，并分阶段补齐可见入口、管理流程、创作绑定与运行闭环。
---

## State

3.1 尚未发布。Stage 1–9 已完成图编辑、运行/调试、自动化创作、基础产品连续性、平台中立 automation seam、Android 产品闭环、Workflow Source portability、资产库规模化/安全清理与 Browser CDP 产品闭环。此前把缺失能力延期到“3.1”的错误边界已撤销；当前唯一未完成的发布前 frontier 是 Stage 10 / Slice 18 的 Source-native 多图创作。

Stage 9 已通过完整 `task check`、production `task build`、正式 Windows WebView smoke，以及真实 Chrome 和 Edge 显式调试端口 smoke。

## Next

执行 Slice 18：定义 graph-call 深模块与 Source annotations/presentation metadata，贯通 schema、compiler、scheduler、journal、debugger、AI authoring、clipboard、portability 和编辑器多图 UX。阶段结束再统一验收与提交。

## Read now

- work/workflow-editor-capability-roadmap/18-source-native-multigraph.md
- work/workflow-editor-capability-roadmap/upgrade-plan.md
- knowledge/architecture/content-addressed-workflow-artifacts.md
- knowledge/build/code-style.md
- knowledge/frontend/ui.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 调整 Stage 10 frontier 或 blocker
- work/workflow-editor-capability-roadmap/artifacts/legacy-product-capability-diff.md — 对照旧能力事实
- work/workflow-editor-capability-roadmap/capability-audit.md — 判断恢复范围或发布阈值
- knowledge/architecture/workspace-file-capability.md — 修改多图 bundle/file 引用边界
- knowledge/subgraph/asset-subsystem.md — 修改资产存储、引用或清理

## Progress

- Stage 1–3 已恢复可靠图编辑、诊断/真调试、模板节点、Blob 预览与键鼠/轨迹录制；均通过阶段门禁。
- Stage 4 已关闭基础 UI、桌面安装/F9、工作流库管理、AI endpoint 与 launcher 回归；Stage 5–6 已完成平台中立 Adapter seam 和 Android/ADB 全闭环。
- Slice 15 已恢复 Source-native Ctrl/⌘+F 节点定位；旧 Container runtime、任意宿主 JS/yt console 继续明确不恢复。
- 2026-07-17 用户纠正发布边界：尚未发布的 3.1 不能通过“post-3.1”延期来宣称完成；Source portability、资产规模化、subgraph/comment 与 Browser CDP 必须发布前实现。
- Slice 16 已完成严格 Workflow Source bundle、copy/replace identity 与列表页 import/export。
- Slice 17 已完成后端资产分页/筛选、跨页批量、variant 管理，以及 asset/Source/Program/Run 全 durable root 的 preview-token Blob cleanup。
- Stage 8 完整 `task check` 通过并提交为 `892b90f9`。
- Slice 19 已完成单一 adapter registry、loopback exact-page profile、Settings discovery/install/consent/health、Browser capability/Catalog/Inspector/模板与组合键语义。
- Stage 9 完整 `task check`、production build、WebView smoke、Chrome smoke 与 Edge smoke 全部通过；smoke PNG 已人工查看。

## Open questions

- 删除工作流时，关联 schedule/launcher item 继续采用引用阻止；历史 Run journal 默认保留，除非后续产品需求明确改变。
- 自定义 AI 地址首期保持 provider-native Responses/Messages，不静默兼容 Chat Completions；该项不阻塞当前 frontier。
