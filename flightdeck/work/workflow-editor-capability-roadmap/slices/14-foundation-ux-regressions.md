---
slice: "14"
title: 基础 UI 与交互回归收口
status: completed
---

# Slice 14：基础 UI 与交互回归收口

## Outcome / Question

把用户已报告且会阻断基础使用的视觉、交互和反馈回归集中关闭，确保新产品壳、工作流首页和编辑器在进入产品能力恢复前已经可理解、可操作且反馈一致。

## Completion criterion

- 主壳、左侧缩放/锁定工具、右侧蓝图预览/小地图及全部浮层完整遵循暗色 token，不残留浏览器默认白底控件。
- 节点标题、字段、输入输出端口和连接点有稳定布局；长 label、缩放、不同端口数量和动态字段下不重叠。
- 全仓浏览器 alert/confirm/prompt 清零，统一使用 Nuxt UI modal/confirm；关闭或离开未保存工作流的确认可键盘操作。
- 成功保存和其他原地动作只做内联状态更新；失败才 toast，运行排队等持续状态优先原地展示，避免成功 toast 噪音。
- 新建工作流可靠包含 Start/run-root，并给出明确的下一步引导。
- 选中节点/边支持 Delete 与 Backspace，输入控件聚焦时不误删；批量删除保持一次 undo。
- 节点目录可搜索、空结果可恢复；拖线候选继续按类型过滤。
- Run 状态只在有实际运行事实时展示，不让每个未运行节点都显示无意义状态；激活窗口等节点清楚解释 target slot 的选择和安装入口。
- 用户报告的单击/连线导致位置跑偏场景重新覆盖，不能因本 Slice 调整布局而回归。

## Blocked by

Stage 3 批量验收；实现前对照当前代码确认哪些问题已由 Slices 1–7 修复，禁止重复重写。

## Verification

先运行相关前端聚合测试和离屏视觉检查；Stage 4 全部实现后统一运行 task check、task build、正式 Windows WebView smoke，并人工检查暗色、端口、键盘删除、退出确认、反馈与 Run 状态。

## Out of scope

品牌视觉重做、主题市场、画布 comment/reroute/subgraph、工作流库分页、平台 target 架构。

## Result

Completed。

- Stage 3 Windows WebView 探针和人工截图已证明 Vue Flow controls/minimap 暗色、端口无重叠、Start 根节点存在、节点无布局重叠且单击/连线位置稳定。
- 全仓原生 alert/confirm/prompt 审计为零；未保存退出继续使用共享 Nuxt UI ConfirmDialog。
- 目录搜索、空结果恢复、类型感知拖线候选和仅按真实 timeline 投影 Run 状态均已存在，不重复重写。
- 增加 edge 选择与 Delete/Backspace 删除；输入、textarea、select、contenteditable 和 dialog 聚焦时不误删，节点批量删除仍是一次命令。
- inspector 展示节点说明；target slot 为空时按 target kind 直达 automation/applications/AI/network 安装设置，使“激活窗口”明确作用于所选 installation slot。
- 工作流首页的 queued/not-started 状态改为行内持续反馈，异常才 toast；保存仍使用工具栏内联“已保存”状态。
- Slice 聚合验证通过：WorkflowAuthoringFoundations + WorkflowsView 共 9 tests、i18n 1499 keys/0 residue、vue-tsc。最终视觉与正式 WebView 交互随 Stage 4 一次性批量验收。
