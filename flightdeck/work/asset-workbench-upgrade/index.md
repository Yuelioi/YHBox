---
topic: asset-workbench-upgrade
title: Asset workbench upgrade
summary: Research and implement a commercial-grade asset browser and management workbench for templates, blueprints, and clips.
---
# Asset workbench upgrade

## State

active

## Next

- 启动最新 bin/Yotta.exe 做 Wails smoke：分别在展开工作台和双击详情打开“安魂曲”，确认两处都立即显示已有 1280×720 变体，并在窗口检测完成后显示当前分辨率与新增变体动作。
- 检查本局收益等低分辨率模板的 1:1 卡片与详情展示。
- 根据下一轮视觉反馈继续收尾资产工作台，稳定后结束本 topic。

## Read now

- knowledge/frontend/ui.md
- knowledge/subgraph/asset-subsystem.md
- knowledge/build/build.md
- knowledge/git/commits.md

## Read if

- knowledge/frontend/async-loading-is-not-unavailable.md — 异步详情把加载中显示成不可用、多个请求互相阻塞或 immediate watcher 初始化异常时
- knowledge/subgraph/keepalive-singleton-subgraph-store-stale.md — 修改模板截图或当前容器上下文时
- knowledge/frontend/headless-ui-verify.md — 需要离屏视觉自检时
- knowledge/frontend/nuxt-ui-icon-button-alignment.md — 修改图标按钮时

## Progress

- 已完成模板、蓝图、Clip 的资产工作台升级：统一浏览工具栏、分类、分页、详情、维护入口、全局搜索与响应式工作区。
- 已将 Node 工具链升级到 24.18.0，并保持仓库完整门禁通过。
- 已将 ContainerEditorView 的 containerID 显式贯穿 AssetDockPanel、TemplateAssetPanel 和两个 TemplateDetailPanel 入口。
- 已把模板 record 与窗口分辨率拆成 recordLoading / resolutionLoading 两条独立链路；已有变体不再等待窗口解析，检测中显示 loading，只有请求结束仍无结果才显示“窗口未开”。
- 已用真实资产确认“本局收益”裁剪尺寸为 96×27；卡片与普通详情改为 1:1 像素上限，大图查看保留 2×。
- 2026-07-14 用户提供生产堆栈 ReferenceError: Cannot access 'editingName' before initialization。根因是 guid 的 immediate watcher 在 setup 中同步执行，却访问了文件后部才声明的 editingName/editingDesc refs；异常中断 detail record 初始化，导致双击详情只剩 skeleton、缺少已有变体。
- 新增 TemplateDetailPanel 真实挂载回归测试：修复前稳定捕获同一 ReferenceError；修复后要求 errorHandler 为空并确认异步 DOM 中出现 1920×1080 变体。编辑状态 refs 已移动到所有 immediate watcher 之前。
- 2026-07-14 最新 task check 明确 exit 0：前端 92 个测试文件、611 项测试通过，Go、lint、类型、i18n、bindings、生产构建和 bundle budget 均通过。
- 最新 task build 明确 exit 0，已重新生成包含本修复的 bin/Yotta.exe。
- 相关提交基线：5f8a5dac、808dffc6、e915af3a、db536810、3afd1647、666f3a64、07bfa1ae、559ebfa6、78c774af、c5bc69ba；本批 TDZ 修复待提交。

## Open questions

- 最新 bin/Yotta.exe 的双击详情与展开详情是否都能稳定显示同一变体内容。
- 1:1 是卡片和普通详情的默认保真上限；后续是否为不同资产类型提供可配置倍率，等待真实素材库反馈后决定。
