---
topic: v3.1-product-optimization
title: 3.1 产品创作体验与运行工作台优化
summary: 在稳定的 3.1 架构上修正编辑器状态表达，恢复专业级多选与子图创作，重新判定调试能力，并重构复杂节点的可理解交互。
---

## State

Stage A completed，进入 Stage B。状态通道、专业画布选择和 Source-native 子图创作已经完成聚合测试、`task check`、真实 Windows WebView smoke 与视觉验收；下一阶段把简易宏和精准录制拆成两个独立领域与产品入口。

## Next

实施 Slice 04：建立原子 MacroAction 文档、验证器、held-input 状态机、宏编辑器和“回放宏”节点；不再把交叠按键压成 grouped keys。

## Read now

- work/v3.1-product-optimization/optimization-plan.md
- work/v3.1-product-optimization/slices/04-atomic-macro-model-and-editor.md
- work/v3.1-product-optimization/context/current-vs-3.0-editor-audit.md

## Read if

- work/v3.1-product-optimization/slices/map.md — 切换 Slice、重排阶段或检查完整实施前沿
- work/workflow-editor-capability-roadmap/slices/03-selection-layout.md — 追溯旧 Topic 对多选、布局和框选的完成声明
- work/workflow-editor-capability-roadmap/slices/18-source-native-multigraph.md — 实施 Slice 03 时复查 Source-native GraphCall 的架构边界
- work/workflow-editor-capability-roadmap/context/user-device-test-2-design.md — 需要资源浏览、调试与专业工作区参考
- knowledge/architecture/feature-continuity-across-product-stack.md — 标记阶段能力完成前复查五层闭环
- knowledge/frontend/vue-flow-store-vmodel-shallow-sync.md — 修改 Vue Flow 选择、拖拽或节点投影
- knowledge/frontend/ui.md — 正式前端实现与视觉验收

## Progress

- 用户已批准一个可见子图入口 + 多个命名出口、调试真机硬闸门和 Authoring Surface 三层模型。
- 规划 Slice 01 已完成；九个已知 Slice 已登记到 slices/map.md。
- Slice 02 与 Slice 03 已完成；选择、执行、调试和校验使用独立视觉通道，画布恢复真实框选与专业手势。
- Vue Flow 瞬时选择由内部 store 拥有，Source 只持久化内容和拖拽结束位置，避免节点位置回跳。
- 子图继续使用 3.1 Source、Compiler 与 Program；authoring 边界只投影入口、命名出口与数据接口，不进入 runtime。
- `task check` 通过；Go 全量门禁、229 个前端测试、类型/格式/静态检查、构建和包体预算为绿。
- Windows WebView smoke 通过真实框选、调试单步、子图导航与边界无遮挡检查；视觉证据位于 `.task/workflow-editor-smoke/20260719-154322/`。
- Stage B 从原子宏开始，精准录制随后独立实施；两者不共享产品资源类型、模式下拉框或编辑器。

## Open questions

- 无。Stage B 按已批准的宏/精准录制分轨执行。
