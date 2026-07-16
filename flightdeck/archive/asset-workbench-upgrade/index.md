---
topic: asset-workbench-upgrade
title: Asset workbench upgrade
summary: Research and implement a commercial-grade asset browser and management workbench for templates, blueprints, and clips.
---
# Asset workbench upgrade

## State

active

## Next

- 启动最新 bin/Yotta.exe 做 Wails smoke：确认普通详情和展开详情里的“新增分辨率变体”均为内容宽度，不再横跨面板。
- 在全局设置的“编辑器显示”切换简洁/完整，确认只影响节点 ID、复制技术信息和节点日志开关；变量、Snippets、调试、JS 控制台、节点改名、Expr 修复建议与输出绑定始终可用。
- 检查本局收益等低分辨率模板的 1:1 卡片与详情展示，根据下一轮视觉反馈继续收尾资产工作台。

## Read now

- knowledge/frontend/ui.md
- knowledge/subgraph/asset-subsystem.md
- knowledge/build/build.md
- knowledge/git/commits.md

## Read if

- knowledge/frontend/async-loading-is-not-unavailable.md — 异步详情把加载中显示成不可用、多个请求互相阻塞或 immediate watcher 初始化异常时
- knowledge/frontend/display-preferences-must-not-gate-capabilities.md — 设计简洁/完整、基础/专业、密度或高级功能显示偏好时
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
- 2026-07-14 将详情面板共享的“新增分辨率变体”动作从 block 全宽按钮改为内容宽度，普通详情与展开详情同步生效，并增加回归检查。
- 深入审计原“基础/专业”模式：它跨工具栏、左侧 rail、Inspector 隐藏多类能力，却又能被快捷键、命令面板和右键菜单绕过；结论是它不是可靠能力边界，而是不一致的显示预设。
- 已移除顶部常驻“基础/专业”入口，统一暴露变量、Snippets、调试、JS 控制台、节点改名、Expr 链建议和输出绑定。全局设置新增“编辑器显示 → 节点信息详细程度（简洁/完整）”，只控制节点 ID、复制技术信息和节点日志开关。
- 2026-07-14 最新 task check 明确 exit 0：前端 93 个测试文件、613 项测试通过，Go、lint、类型、i18n、bindings、生产构建和 bundle budget 均通过。
- 最新 task build 明确 exit 0，已重新生成包含本轮 UI 改造的 bin/Yotta.exe。
- 相关提交：dde4a7dc（紧凑变体动作）、5cfd1b09（编辑器显示偏好重构）。

## Open questions

- 最新 bin/Yotta.exe 的普通详情与展开详情是否都达到预期的紧凑按钮宽度。
- 简洁/完整现在仅表达节点技术信息密度；是否继续拆出独立的工具栏密度偏好，等待真实编辑使用反馈后决定。
- 1:1 是卡片和普通详情的默认保真上限；后续是否为不同资产类型提供可配置倍率，等待真实素材库反馈后决定。
