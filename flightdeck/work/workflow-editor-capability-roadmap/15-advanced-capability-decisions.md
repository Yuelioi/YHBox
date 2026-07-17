---
slice: "15"
title: 高级能力决策与迁移收口
status: in_progress
---

# Slice 15：高级能力决策与迁移收口

## Outcome / Question

恢复旧版中仍具产品价值的高级能力，并把确实不应恢复的旧模型替换为 3.1 Source-native 设计；在 3.1 尚未发布时，不允许用“未来版本”代替实现。

## Completion criterion

- Source-native 节点定位完成。
- Slices 16–19 全部完成：Source portability、资产规模化、Browser CDP 产品闭环、subgraph/comment/reroute。
- 旧 Container runtime 与任意宿主脚本明确不恢复，并有安全替代。
- Stage 8–10 分别完成批量验收，3.1 发布阈值没有未授权延期项。

## Blocked by

无。Slices 16 与 17 可顺序执行；Slice 19 复用 Stage 5 Adapter seam；Slice 18 在 portability 稳定后执行。

## Verification

以 Slices 16–19 各自完成标准和 Stage 8–10 批量门禁为准，最终再运行 Flightdeck check。

## Out of scope

复制旧 Container runtime、恢复任意宿主 JS/Wails/yt console、为兼容旧 UI 创建第二套执行器。

## Result

Reopened。之前仅完成节点定位和能力决策，却把仍在 3.1 范围内的能力标为 post-3.1 并关闭，这是错误的完成定义。当前以 commit aaa34711 为恢复基线，待 Slices 16–19 全部交付后再完成本 Slice。
