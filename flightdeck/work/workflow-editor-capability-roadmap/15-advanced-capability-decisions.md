---
slice: "15"
title: 高级能力决策与迁移收口
status: completed
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

无。

## Verification

以 Slices 16–19 各自完成标准和 Stage 8–10 批量门禁为准，最终运行 Flightdeck check。

## Out of scope

复制旧 Container runtime、恢复任意宿主 JS/Wails/yt console、为兼容旧 UI 创建第二套执行器。

## Result

Completed。Source-native 节点定位、严格 Workflow Source portability、资产规模化/安全清理、Browser CDP 产品闭环与 Source-native 多图创作均已交付。旧 Container runtime 不进入 3.1 Source；脚本能力只允许走 capability-admitted 隔离执行。Stage 8–10 均完成批量门禁，未把本版本范围内能力擅自延期。
