---
topic: workflow-authoring-foundations
title: 工作流创作基础与旧能力连续性
summary: 让 3.1 工作流从新建即可理解和操作，并核清/接回录制、Clips、图片模板等旧能力在新执行链路中的产品入口。
---

## State

目标已达成：3.1 工作流具备新建起点、目录搜索、键盘删除、工作流状态、exact target 选择与独立资源库；录制、Clip 和视觉模板的主窗口入口已恢复。

## Next

无。本 Topic 完成后归档；WaitTemplate / WaitTemplateGone / ClickTemplate 复合节点与安全模板缩略图 adapter 作为独立后续工作，不留在本 Topic。

## Read now

- knowledge/agent/codex-working-agreement.md
- knowledge/build/code-style.md
- knowledge/frontend/ui.md
- knowledge/frontend/display-preferences-must-not-gate-capabilities.md
- knowledge/architecture/installed-input-authority.md
- knowledge/architecture/feature-continuity-across-product-stack.md
- archive/workflow-authoring-foundations/slices/authoring-basics.md
- archive/workflow-authoring-foundations/slices/resource-continuity.md

## Read if

- knowledge/build/build.md — 开始阶段末编译、测试、构建或真实 GUI smoke
- knowledge/frontend/headless-ui-verify.md — 进行离屏前端视觉检查
- knowledge/nodes/recording-schema-cascade.md — 修改录制节点 schema 或录制保存链路
- knowledge/input/ll-hook-keydown-coalesce.md — 修改低级键盘或鼠标 hook 停录行为
- knowledge/frontend/nuxt-ui-icon-button-alignment.md — 新增或修改 icon-only UButton
- knowledge/frontend/vue-flow-delete-key-code-ignores-modifiers.md — 实现 Vue Flow Delete/Backspace 删除

## Progress

- 已确认录制器、HUD、InputClip/asset store、PlayInputClip、CaptureWindow、MatchTemplate 与 FindTemplateMatches runtime 均保留。
- 已确认旧 Container 产品栈删除时一并移除了主窗口录制与资源管理 UI，问题是 3.1 产品迁移未完成而非底层能力全部删除。
- 新建 Workflow Source 由后端权威注入 RunStarted 根节点；旧空图仍有画布引导恢复入口。
- 节点目录已增加搜索与分类，普通 Delete/Backspace 通过 EditorCommand 删除选中节点。
- Run 状态已移出节点 Inspector；exact target 配置字段从设置中心安装项选择，高级能力/状态信息渐进披露。
- 独立资源库提供录制开始/暂停/完成/入库、Clip 与模板搜索、元数据编辑、删除和模板截图入口。
- 阶段末 `task check` 全绿：全局 Go 覆盖率 65.1%，vet/staticcheck、前端格式/lint/typecheck/i18n/Vitest 与生产 bundle 全部通过。
- 真实 Windows Wails WebView smoke 通过：100 个目录节点、新建 RunStarted、搜索/节点增删/状态与 AI 面板、资源库/录制入口均可达，无 JS error/rejection/console.error；编辑器与资源库截图人工检查通过。

## Open questions

无阻塞问题。旧 WaitTemplate、WaitTemplateGone、ClickTemplate 复合节点和模板实际缩略图安全 adapter 已明确移出本 Topic。
