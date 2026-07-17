---
topic: workflow-editor-capability-roadmap
title: 3.1 产品能力连续性审计与升级路线
summary: 审计旧产品能力与 3.1 现状，按唯一新架构、用户体验和必要性决定恢复、重做或删除，并完成发布前能力补齐。
---

## State

Completed。3.1 尚未发布，但本 Topic 定义的 major upgrade 发布前能力范围已经完成：Stage 1–10 覆盖图编辑、运行/真调试、自动化创作、基础产品连续性、平台中立 automation seam、Windows/Android/Browser 产品闭环、Workflow Source portability、资产规模化/安全清理，以及 Source-native 多图创作。

Stage 10 已完成 graph-call、typed graph interface、递归/深度保护、compiler/scheduler/journal/debug graph path、subgraph 创作、collapse selection、comment annotation、edge reroute、clipboard/布局、AI/MCP authoring 与跨图运行定位。没有把本版本范围内能力延期到所谓“后续 3.1”。

## Next

进入发布准备时，只处理发布工程事项（签名、安装包、release notes、最终真机矩阵）；不再把本 Topic 已完成的产品能力重新解释成未来版本工作。

## Read now

- work/workflow-editor-capability-roadmap/upgrade-plan.md
- work/workflow-editor-capability-roadmap/15-advanced-capability-decisions.md
- work/workflow-editor-capability-roadmap/18-source-native-multigraph.md

## Read if

- work/workflow-editor-capability-roadmap/slices/map.md — 查询各 Slice 最终状态
- work/workflow-editor-capability-roadmap/artifacts/legacy-product-capability-diff.md — 对照旧能力取舍
- work/workflow-editor-capability-roadmap/capability-audit.md — 查询恢复范围和发布阈值
- knowledge/architecture/content-addressed-workflow-artifacts.md — 修改 Workflow/Node durable identity
- knowledge/build/build.md — 进入发布、打包或真机 smoke

## Progress

- Stage 1–3 恢复可靠图编辑、类型感知连线、选择/布局、诊断/真调试、模板节点、Blob 预览与键鼠/轨迹录制。
- Stage 4 关闭暗色、端口、alert/toast、Start/Delete、桌面安装/F9、工作流库、AI endpoint 与 launcher 回归。
- Stage 5–6 完成平台中立 Adapter seam 和 Android/ADB 安装、创作、运行闭环。
- Stage 8 完成严格 Workflow Source bundle、copy/replace identity、资产分页/筛选/跨页批量/variant，以及完整 durable-root Blob cleanup。
- Stage 9 完成 Browser CDP exact installation、Settings discovery/install/consent/health、provider/Catalog/Inspector/模板和 Chrome/Edge smoke。
- Stage 10 完成 Source-native 多图 schema、authoring、compiler/runtime/debugger、编辑器与 MCP 闭环；旧 Container runtime、任意宿主 JS/Wails/yt console 明确不恢复。
- Stage 10 门禁：完整 `task check` 通过（global coverage 65.1%）；production `task build` 通过；Windows WebView 多图 authoring/debug/assets smoke 通过；最终截图已人工检查。
- WebView smoke 证据：`.task/workflow-editor-smoke/20260718-013815/workflow-editor.png` 与 `assets.png`。

## Open questions

- 删除工作流时，关联 schedule/launcher item 继续采用引用阻止；历史 Run journal 默认保留，除非后续产品需求明确改变。
- 自定义 AI 地址首期保持 provider-native Responses/Messages，不静默兼容 Chat Completions。
- 对外发布前仍需按 release 流程决定签名、安装包与许可证表述；当前 LICENSE 是 source-available，不能称 OSI open source。
