---
slice: "18"
title: Source-native subgraph、comment 与 reroute
status: completed
---

# Slice 18：Source-native subgraph、comment 与 reroute

## Outcome / Question

把 Source 的 graphs 结构变成真正可创作、可编译、可运行和可调试的多图能力，并以非执行投影恢复 comment/reroute。

## Completion criterion

- graph-call 深模块定义 typed inputs/outputs、调用节点、引用删除、递归检测和运行深度预算。
- compiler/program/scheduler/journal 支持 graph path；断点、单步、诊断和节点定位不绕过唯一执行路径。
- 编辑器支持创建、进入、返回、重命名、删除 subgraph，并可把选择折叠成 subgraph，一个 undo。
- comment 是 authoring-only annotation；reroute 是 edge presentation metadata，二者不进入 Catalog、capability 或 scheduler。
- clipboard、布局、导入导出和 AI/MCP authoring patch 对多 graph/annotation 数据有明确语义。
- 旧 Container subgraph 格式不直接写入 3.1 Source。

## Blocked by

无；Slice 16 portability 已完成。

## Verification

schema/authoring/compiler/scheduler/debug/editor 聚合覆盖；Stage 10 批量运行 `task check`、`task build` 与 Windows WebView 多图 smoke。

## Out of scope

任意递归无预算执行、旧 Container runtime、把 comment/reroute 注册为执行节点。

## Result

Completed。

- Workflow Source 增加 typed graph ports、GraphCall、Annotation 与 edge presentation reroutes；schema 拒绝未知 callee、非法边界、调用环和超深调用。
- compiler 在编译期递归展开调用图，Program、capability attribution、scheduler、journal 与 debugger 保存 source graph path/provenance，仍使用唯一执行路径。
- authoring engine 提供 graph/interface/call/annotation/reroute/collapse 等命令，并支持同批次新节点 handle 在 graph boundary 中安全解析。
- 编辑器提供子图创建/导航/重命名/删除、调用插入与 typed bindings、接口推断、选择折叠、comment、reroute、clipboard、布局和跨图运行定位。
- MCP inspection 支持图分页与 call/annotation 数据，authoring patch 支持新增命令。
- 完整 `task check` 通过，global coverage 65.1%；production `task build` 通过。
- Windows WebView smoke 完成创建工作流、真调试、连线候选、二次保存、子图接口/调用、comment、reroute、AI 面板和资源库旅程；截图已人工检查。
