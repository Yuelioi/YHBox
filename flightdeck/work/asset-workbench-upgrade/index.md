---
topic: asset-workbench-upgrade
title: Asset workbench upgrade
summary: Research and implement a commercial-grade asset browser and management workbench for templates, blueprints, and clips.
---
# Asset workbench upgrade

## State

active

## Next

- 在 Wails 桌面应用内 smoke：分别从紧凑侧栏和展开工作台打开同一模板，确认当前窗口分辨率、已有变体、新增/重拍操作完全一致。
- 用低分辨率模板检查卡片、详情和大图预览，确认图片最多放大到原始像素的 2 倍且在舞台内居中。
- 根据下一轮视觉反馈继续收尾资产工作台，稳定后结束本 topic。

## Read now

- knowledge/frontend/ui.md
- knowledge/subgraph/asset-subsystem.md
- knowledge/build/build.md
- knowledge/git/commits.md

## Read if

- knowledge/subgraph/keepalive-singleton-subgraph-store-stale.md — 修改模板截图或当前容器上下文时
- knowledge/frontend/headless-ui-verify.md — 需要离屏视觉自检时
- knowledge/frontend/nuxt-ui-icon-button-alignment.md — 修改图标按钮时

## Progress

- 已完成模板、蓝图、Clip 的资产工作台升级：统一浏览工具栏、分类、分页、详情、维护入口、全局搜索与响应式工作区。
- 已将 Node 工具链升级到 24.18.0，并保持仓库完整门禁通过。
- 已修复模板详情的变体管理可见性，并把分辨率变体前置到核心信息区。
- 已修复紧凑侧栏与展开工作台的功能漂移：ContainerEditorView 现在把明确的 containerID 贯穿 AssetDockPanel、TemplateAssetPanel 和两个 TemplateDetailPanel 入口；当前分辨率和重拍不再依赖模板 store 的隐式全局时序。
- 已新增 CappedPreviewImage，卡片、详情和大图预览统一按图片 naturalWidth/naturalHeight 限制最大 2 倍放大，避免低分辨率模板被无限插值。
- 已补充容器上下文透传和像素倍率回归测试；2026-07-14 的 task check 全通过，前端 91 个测试文件、609 项测试通过，Go、lint、类型、i18n、bindings、生产构建和 bundle budget 均通过。
- 相关提交基线：5f8a5dac、808dffc6、e915af3a、db536810、3afd1647、666f3a64、07bfa1ae；本批修复待提交。

## Open questions

- 真实目标窗口在紧凑侧栏与展开工作台之间切换时，Wails smoke 是否都能稳定返回同一 currentResolution。
- 2 倍是当前默认保真上限；后续是否需要允许用户在设置中调整，等待真实素材库反馈后再决定。
