---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做、延期或删除，并分阶段补齐可见入口、管理流程、创作绑定与运行闭环。
---

## State

3.1 尚未发布。Stage 1–8 已完成图编辑、运行/调试、自动化创作、基础产品连续性、平台中立 automation seam、Android 产品闭环、Workflow Source portability 与资产库规模化/安全清理。原 Stage 7 对仍缺能力的错误延期已撤销，Browser CDP 与 source-native 多图创作仍在 3.1 发布前范围。

Stage 8 已通过完整 `task check`。当前 frontier 是 Stage 9 / Slice 19：把已有 Browser CDP 底层 controller 接入 exact installation、Settings、Catalog、provider 与真实浏览器 smoke。

## Next

执行 Slice 19：审计现有 browsercdp 与 automation registry seam，完成 loopback exact-page profile、发现/安装/consent/健康检查、browser targetKinds 节点开放和真实 Chrome/Edge 调试端口 smoke。阶段结束再统一验收与提交。

## Read now

- work/workflow-editor-capability-roadmap/19-browser-cdp-installation.md
- work/workflow-editor-capability-roadmap/upgrade-plan.md
- knowledge/architecture/go-multiplatform-boundary.md
- knowledge/build/code-style.md
- knowledge/frontend/ui.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 调整 Stage 9–10 frontier 或 blocker
- work/workflow-editor-capability-roadmap/artifacts/legacy-product-capability-diff.md — 对照旧能力事实
- work/workflow-editor-capability-roadmap/capability-audit.md — 判断恢复范围或发布阈值
- knowledge/architecture/content-addressed-workflow-artifacts.md — 修改 Source/bundle/hash/identity
- knowledge/architecture/workspace-file-capability.md — 设计导入导出文件边界
- knowledge/subgraph/asset-subsystem.md — 修改资产存储、引用或清理

## Progress

- Stage 1–3 已恢复可靠图编辑、诊断/真调试、模板节点、Blob 预览与键鼠/轨迹录制；均通过阶段门禁。
- Stage 4 已关闭基础 UI、桌面安装/F9、工作流库管理、AI endpoint 与 launcher 回归；Stage 5–6 已完成平台中立 Adapter seam 和 Android/ADB 全闭环。
- Slice 15 已恢复 Source-native Ctrl/⌘+F 节点定位；旧 Container runtime、任意宿主 JS/yt console 继续明确不恢复。
- 2026-07-17 用户纠正发布边界：尚未发布的 3.1 不能通过“post-3.1”延期来宣称完成；Source portability、资产规模化、subgraph/comment 与 Browser CDP 必须发布前实现。
- Slice 16 已完成严格 Workflow Source bundle、copy/replace identity 与列表页 import/export。
- Slice 17 已完成后端资产分页/筛选、跨页批量、variant 管理，以及 asset/Source/Program/Run 全 durable root 的 preview-token Blob cleanup。
- Stage 8 完整 `task check` 通过；第一次门禁遇到 Windows 测试文件临时占用，定向重跑通过；第二次发现并修复 ST1005；最终完整门禁退出 0。

## Open questions

- 删除工作流时，关联 schedule/launcher item 继续采用引用阻止；历史 Run journal 默认保留，除非后续产品需求明确改变。
- 自定义 AI 地址首期保持 provider-native Responses/Messages，不静默兼容 Chat Completions；该项不阻塞当前 frontier。
