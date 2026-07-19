---
topic: v3.1-product-optimization
title: 3.1 产品创作体验与运行工作台优化
summary: 在稳定的 3.1 架构上修正编辑器状态表达，恢复专业级多选与子图创作，重新判定调试能力，并重构复杂节点的可理解交互。
---

## State

Stage G completed。Slice 11 已用用户原始症状的 trusted WebView 红灯修复完整画布缩放；Slice 12 已删除硬编码“常用配方”，并以 3.1 durable application service 恢复用户自定义 Snippets。

## Next

由用户在普通 UAC 开发构建中复核空白画布缩放与 Snippet 保存/插入手感；继续下一项产品优化前先选择并登记新的 Slice。

## Read now

- work/v3.1-product-optimization/optimization-plan.md
- work/v3.1-product-optimization/slices/12-snippets-restoration.md
- knowledge/architecture/feature-continuity-across-product-stack.md

## Read if

- work/v3.1-product-optimization/slices/map.md — 重排阶段、选择 Slice 12 或检查完整实施前沿
- work/v3.1-product-optimization/slices/11-canvas-node-density-and-wheel-zoom.md — 复查完整画布 wheel 红灯与修复证据
- work/v3.1-product-optimization/context/current-vs-3.0-editor-audit.md — 复查 3.0 编辑器经验
- knowledge/architecture/feature-continuity-across-product-stack.md — Snippets 标记完成前复查五层闭环
- knowledge/frontend/ui.md — 正式前端实现与视觉验收
- knowledge/build/code-style.md — Go/TypeScript 实现与生成契约

## Progress

- Stage A–C 已恢复专业画布选择、Source-native 子图、Macro/InputClip、编辑器资源工作区和真实调试链路。
- Stage D–E 已建立 typed Authoring Surface、复杂 Editor Adapter、视觉能力和黄金旅程。
- Slice 11 已修复节点膨胀与完整 wheel ownership；trusted WebView 红灯先复现空白画布 2.000→2.000，再由统一外层 handler 修复，空白和未选中节点均通过。
- Slice 12 已恢复用户保存“带配置节点”的 Snippets：名称、描述、分类、标签和使用时间均 durable 持久化，支持搜索筛选、编辑删除、点击/拖放插入；硬编码“常用配方”及其 builder 已删除。
- Stage G 分成两个独立 Slice：11 修正完整画布 wheel ownership，12 删除 recipes 并用 3.1 Workflow Source/Node Contract 恢复 Snippets。
- Stage G 已通过 `task check` 和真实 Wails/WebView smoke；smoke 产物位于 `.task/workflow-editor-smoke/20260719-235101/`。

## Open questions

- 无。
