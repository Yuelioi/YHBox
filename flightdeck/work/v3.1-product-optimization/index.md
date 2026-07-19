---
topic: v3.1-product-optimization
title: 3.1 产品创作体验与运行工作台优化
summary: 在稳定的 3.1 架构上修正编辑器状态表达，恢复专业级多选与子图创作，重新判定调试能力，并重构复杂节点的可理解交互。
---

## State

Stage H completed。Slice 13 已把节点右键升级为标准命令菜单，并把视觉模板选择、截图入口和 typed Asset binding 纳入当前工作流内的连续任务流。

## Next

由用户在普通 UAC 开发构建中复核节点菜单手感，并用真实自动化目标从“视觉模板 → 截图新模板”完成一次宿主截图；继续产品优化前选择并登记新的 Slice。

## Read now

- work/v3.1-product-optimization/slices/13-node-context-menu-and-template-flow.md
- knowledge/architecture/feature-continuity-across-product-stack.md

## Read if

- work/v3.1-product-optimization/slices/map.md — 检查完整实施前沿或选择下一 Slice
- work/v3.1-product-optimization/slices/12-snippets-restoration.md — 复查 Snippet payload、持久化和插入契约
- work/v3.1-product-optimization/context/current-vs-3.0-editor-audit.md — 复查 3.0 编辑器经验
- knowledge/frontend/ui.md — 修改菜单、资源工作区或模板 UI
- knowledge/build/code-style.md — Go/TypeScript 实现与生成契约

## Progress

- Stage A–C 已恢复专业画布选择、Source-native 子图、Macro/InputClip、编辑器资源工作区和真实调试链路。
- Stage D–E 已建立 typed Authoring Surface、复杂 Editor Adapter、视觉能力和黄金旅程。
- Stage F–G 已修复节点密度、完整画布 wheel ownership，并删除硬编码配方、恢复 durable Snippets。
- Slice 13 已以受控 Nuxt UI 菜单恢复复制、剪切、复制节点、启停、断点、折叠子图、视觉模板、Snippet 和删除；右键选择语义兼容多选。
- 视觉模板可从节点菜单直接打开编辑器内模板资源分页或发起截图；兼容节点绑定 BlobRef，不兼容节点沿既有契约插入 click-template。
- Stage H 的真实 WebView 旅程与最终前端聚合门禁通过；完整 `task check` 仅被已知高负载 timer 上限 flaky 阻断，定向测试通过。

## Open questions

- 无。
