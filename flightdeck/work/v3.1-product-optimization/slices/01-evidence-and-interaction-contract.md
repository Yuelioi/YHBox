---
slice: "01"
title: 编辑器优化事实基线与交互契约
status: completed
---

## Outcome / Question

为独立的 3.1 产品优化 Topic 建立可信事实基线，并提交一份可以批准、调整或否决的总方案。回答当前问题属于架构缺陷、产品表达缺陷还是实现回归，明确哪些 3.0 交互应恢复、哪些旧架构不得回接，并定义阶段实施与批量验收边界。

## Completion criterion

- 新 Topic 与旧“升级到 3.1” Topic 的职责边界写清楚。
- 当前源码、3.0 reference 和已有本地研究完成第一轮差异审计。
- optimization-plan.md 包含架构判断、产品原则、阶段、Slices、验收闸门和非目标。
- 简易宏采用原子动作与显式 Sleep，精准录制采用原始 InputClip 的分轨方案写入总方案。
- 用户明确批准子图、调试和复杂节点三项关键设计。

## Blocked by

无。用户已批准总方案与三项关键设计。

## Verification

- 对照 WorkflowNode 与 WorkflowEditorView 确认状态和框选问题。
- 对照 workflow-source、EditorSession 和 compiler graph expansion 确认子图真实语义。
- 对照 Application、DebugController、EditorSession 与用户真机反馈确认调试尚未闭环。
- 对照 ColorRangeValueEditor、ScreenPicker 和 ExtractColorRange 确认复杂节点断点。
- 对照 InputClip model、recording canonicalize、draft conversion 和 RecordingActionEditor 确认 grouped keys 的有损转换。
- 对照 yotta-3.0-reference 的 virtual markers、PinLiteral 和工作区布局提取可保留交互。
- flightdeck_check 已通过；本 Slice 没有运行产品门禁。

## Out of scope

- Go、Vue、TypeScript、CSS、schema 或生成物修改。
- 完整测试、production build 或 UAC 真机 smoke。
- 回滚 3.1 或恢复旧 Container runtime。

## Result

用户已批准五阶段、九 Slice 的总方案，以及一个可见子图入口 + 多个命名出口、调试真机硬闸门、通用投影底座 + 类型级 Editor Adapter + 任务配方三项设计。产品代码仍未修改，实施进入阶段 A。
