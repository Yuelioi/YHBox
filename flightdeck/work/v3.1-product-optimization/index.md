---
topic: v3.1-product-optimization
title: 3.1 产品创作体验与运行工作台优化
summary: 在稳定的 3.1 架构上修正编辑器状态表达，恢复专业级多选与子图创作，重新判定调试能力，并重构复杂节点的可理解交互。
---

## State

Stage A 至 Stage D 已完成。Authoring Surface 深模块、复杂类型 Editor Adapter 与第一条视觉分析配方闭环已落地，当前进入最终 Slice 09 工作区一致性与黄金路径收口。

## Next

启动 Slice 09：以工作流创建、目标绑定、资源创建/选择、编译运行、调试和计划执行为连续黄金路径，完成 3.1 发布前的一致性与商业交付收口。

## Read now

- work/v3.1-product-optimization/optimization-plan.md
- work/v3.1-product-optimization/slices/09-workspace-consistency-golden-path.md

## Read if

- work/v3.1-product-optimization/slices/map.md — 重排阶段、选择后续 Slice 或检查完整实施前沿
- work/v3.1-product-optimization/context/current-vs-3.0-editor-audit.md — 复查 3.0 复杂输入和节点卡片经验
- work/workflow-editor-capability-roadmap/context/user-device-test-2-design.md — 复查既有资源浏览和专业工作区设计
- knowledge/architecture/feature-continuity-across-product-stack.md — 标记复杂节点能力完成前复查五层闭环
- knowledge/frontend/ui.md — 正式前端实现与视觉验收
- knowledge/build/code-style.md — Go/TypeScript 实现与生成契约

## Progress

- Stage A 已在 9017af9f3da1ef004d732ebf28a36d4e14dc3a7f 提交，框选、状态通道和 Source-native 子图创作保留。
- Stage B/C 恢复 Macro、InputClip、录制保存、编辑器资源工作区和真实多节点调试链路，并在 6822c87c 提交。
- Stage D 源码审计确认执行内核健康，主要缺口在 Authoring Projection：当前只有基本 control/constraints/editorAdapter，页面仍直接分支复杂控件，缺少分组、单位、重要性、内联优先级和截图拾取上下文。
- 3.0 可借鉴的是 Point/Region/Color 的任务控件、结构/高级模式和 ScreenPicker 闭环；不得复制旧 Container runtime、kind 分发或脚本字面量系统。
- 当前 ScreenPicker 与 ExtractColorRange 已支持目标截图、点/矩形/颜色结果，Stage D 只需把它们通过通用 adapter 接回 typed value。
- Stage D 已把 presentation metadata、统一 Authoring Surface、Point/Region/Duration/KeyChord/Asset/Target adapter、渐进 Inspector 和节点内联高频输入落地。
- ColorRange 现支持色样摘要、RGB/HSV 高级通道与目标取色；视觉配方只生成普通 Capture/Vision/Comparison/Collection/Branch 节点与 typed edges。
- 阶段验收通过 `task check`、production `task build` 与隔离 WebView2 CDP smoke；复杂编辑器按需加载后初始 editor gzip 212,537 bytes，低于 220,000 上限。

## Open questions

- 无。
