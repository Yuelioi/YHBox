---
topic: asset-workbench-upgrade
title: Asset workbench upgrade
summary: Research and implement a commercial-grade asset browser and management workbench for templates, blueprints, and clips.
---
# Asset workbench upgrade

## State

active

## Next

- 在新构建的 Wails 桌面应用内 smoke：打开“容器 4”的模板详情，确认检测期间显示加载状态，已有 1920×1080 变体立即出现，检测完成后显示当前窗口分辨率。
- 检查本局收益等 96×27 低分辨率模板：卡片与普通详情保持 1:1 像素上限，大图查看允许最多 2×。
- 根据下一轮视觉反馈继续收尾资产工作台，稳定后结束本 topic。

## Read now

- knowledge/frontend/ui.md
- knowledge/subgraph/asset-subsystem.md
- knowledge/build/build.md
- knowledge/git/commits.md

## Read if

- knowledge/frontend/async-loading-is-not-unavailable.md — 异步详情把加载中显示成不可用或多个请求互相阻塞时
- knowledge/subgraph/keepalive-singleton-subgraph-store-stale.md — 修改模板截图或当前容器上下文时
- knowledge/frontend/headless-ui-verify.md — 需要离屏视觉自检时
- knowledge/frontend/nuxt-ui-icon-button-alignment.md — 修改图标按钮时

## Progress

- 已完成模板、蓝图、Clip 的资产工作台升级：统一浏览工具栏、分类、分页、详情、维护入口、全局搜索与响应式工作区。
- 已将 Node 工具链升级到 24.18.0，并保持仓库完整门禁通过。
- 已修复模板详情的变体管理可见性，并把分辨率变体前置到核心信息区。
- 已将 ContainerEditorView 的 containerID 显式贯穿 AssetDockPanel、TemplateAssetPanel 和两个 TemplateDetailPanel 入口。
- 2026-07-14 针对用户最新截图完成真实数据诊断：容器 4 的 templateCaptureAdapter 能成功解析当前窗口；截图中的“窗口未开”与变体 skeleton 同时出现，根因是 pending 的 curRes=null 被错误渲染成 unavailable，并且 backend.assets.get 与最长 3 秒的 currentResolution 被绑进同一个 Promise.allSettled。
- 已把模板 record 与窗口分辨率拆成 recordLoading / resolutionLoading 两条独立链路；已有变体不再等待窗口解析，检测中显示 loading，只有请求结束仍无结果才显示“窗口未开”，containerId 变化会重新检测。
- 已用真实资产确认“本局收益”裁剪尺寸为 96×27；139px 卡片只放大约 1.45×，原先 2×阈值不会触发。卡片与普通详情现改为 1:1 像素上限，大图查看保留 2×。
- 回归命令 pnpm vitest run src/components/containers/InteractionAccessibility.spec.ts 在修复前稳定失败 2 项，修复后聚焦 23 项通过。
- 2026-07-14 最新 task check 全通过：前端 91 个测试文件、610 项测试通过，Go、lint、类型、i18n、bindings、生产构建和 bundle budget 均通过。
- 相关提交基线：5f8a5dac、808dffc6、e915af3a、db536810、3afd1647、666f3a64、07bfa1ae、559ebfa6；本批 pending/unavailable 与 1:1 预览修复待提交。

## Open questions

- 新构建中窗口检测完成后的 currentResolution 是否与真实容器诊断一致；若 Wails RPC 最终仍失败，需要保留错误原因而不是由 backend wrapper 静默吞掉。
- 1:1 是卡片和普通详情的默认保真上限；后续是否为不同资产类型提供可配置倍率，等待真实素材库反馈后决定。
