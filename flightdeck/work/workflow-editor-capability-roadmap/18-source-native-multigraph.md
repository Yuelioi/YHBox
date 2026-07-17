---
slice: "18"
title: Source-native subgraph、comment 与 reroute
status: pending
---

# Slice 18：Source-native subgraph、comment 与 reroute

## Outcome / Question

把 Source 已有的 graphs 结构变成真正可创作、可编译、可运行和可调试的多图能力，并以非执行投影恢复 comment/reroute。

## Completion criterion

- graph-call 深模块定义 typed inputs/outputs、调用节点、引用删除、递归检测和运行深度预算。
- compiler/program/scheduler/journal 支持 graph path；断点、单步、诊断和节点定位不绕过唯一执行路径。
- 编辑器支持创建、进入、返回、重命名、删除 subgraph，并可把选择折叠成 subgraph，一个 undo。
- comment 是 authoring-only annotation；reroute 是 edge presentation metadata，二者不进入 Catalog、capability 或 scheduler。
- clipboard、布局、导入导出和 AI authoring patch 对多 graph/annotation 数据有明确语义。
- 旧 Container subgraph 格式不直接写入 3.1 Source。

## Blocked by

Slice 16 的完整 Source portability；现有 compiler 单 main graph 限制需要在本 Slice 内移除。

## Verification

schema/authoring/compiler/scheduler/debug/editor 聚合测试；Stage 10 批量运行 task check、task build 与 Windows WebView 多图 smoke。

## Out of scope

任意递归无预算执行、旧 Container runtime、把 comment/reroute 注册为执行节点。

## Result

Pending。
