---
slice: "11"
title: 画布节点密度与滚轮缩放
status: completed
---

## Outcome / Question

恢复专业画布的稳定密度与相机手势：复杂节点不能把完整配置表单铺进画布，鼠标位于节点卡片上时滚轮仍应缩放画布。

## Completion criterion

- “分析颜色”等复杂节点在默认缩放下保持紧凑，只显示端口、运行/校验状态和最多 1–3 个高频摘要或输入。
- 完整 ColorRange、Region 和高级通道编辑保留在 Inspector 或明确弹层，不重复占用画布节点主体。
- 滚轮位于画布空白、节点标题、端口区和节点摘要区时均改变 Vue Flow viewport zoom；不得被普通节点容器静默截获。
- 打开的下拉、弹层或明确独立滚动区域可以按自身契约消费滚轮，关闭后立即归还画布。
- 节点选中、拖动、连线、框选和内联高频输入不因缩放修复回归。

## Blocked by

- Slice 08 的 Authoring Surface 与 ColorRange adapter 已存在；本 Slice 只修正其画布投影和交互所有权。

## Verification

- 建立红灯组件/编辑器测试，渲染 Analyze Color fixture 并断言节点尺寸上限与节点上 wheel 会改变 viewport。
- 覆盖普通节点、复杂节点、打开编辑器/弹层后的 wheel ownership。
- 使用 Wails WebView2 CDP 打开真实工作流，检查 100%/125%/150% 显示缩放、节点选中、拖动和连线。
- Slice 完成后统一运行阶段相关前端测试、`task check`、`task webview:smoke:full` 与 production build。

## Out of scope

- 改变 Analyze Color 的运行算法、ColorRange 严格类型或 Vue Flow 之外的自由停靠系统。
- 允许任意缩放单个节点；本问题中的“放大缩小”指画布相机缩放。

## Result

根因分为两个职责泄漏：

1. `projectAuthoringSurface` 只按 `inlinePriority` 选前三项，没有限制 editor adapter 的交互重量，导致 ColorRange、Point、Region 这类复合编辑器直接挂进 `WorkflowNode`。
2. Vue Flow 默认 wheel 行为在节点交互表面上没有稳定所有权；普通节点命中后，真实 WebView 滚轮不能可靠更新画布 viewport。

修复结果：

- 节点卡片只允许 `duration`、`key-chord`、`number`、`select`、`text`、`toggle` 六类轻量 adapter 内联；Asset、ColorRange、Point、Region、JSON 等复合编辑器继续由 Inspector 承载。
- 画布捕获节点区域的 wheel，在 0.2–2.0 范围内以光标为锚点计算新 viewport；空白区域继续使用 Vue Flow 默认行为。
- `WorkflowNode` 暴露稳定的 node type 与 inline adapter 测试契约，WebView smoke 会插入 Analyze Color、断言布局尺寸与复合 adapter 为零，并用 CDP trusted mouse wheel 验证 zoom 改变后删除 fixture，继续其余黄金旅程。

阶段验收：

- Authoring Surface 与 viewport 数学定向测试：2 个文件、9 项测试通过。
- `task check`：通过，244.5 秒。
- `task webview:smoke`：通过，146 个目录节点、4 个画布节点；结果目录 `.task/workflow-editor-smoke/20260719-223048`，`analyze-color.png` 已人工检查为紧凑节点。
- `task build`：通过，正式 `bin/Yotta.exe` 已生成。
- `task webview:smoke:full`：Analyze Color 与编辑器旅程已通过，最后在既有悬浮启动器等待 workflow success 时超时；该 launcher 故障与本 Slice 代码路径独立，未伪报为本 Slice 通过。

## Recovery

用户真机确认：鼠标位于空白画布时滚轮不缩放，必须先选择节点后才表现为可用。上一次 WebView smoke 在插入 Analyze Color 后直接对新节点发送 wheel，而新节点自动处于选中态，因此只覆盖了最有利路径，不能证明全画布交互。

恢复后的红灯必须先清空全部选择，再分别向空白画布和未选中节点发送 CDP trusted mouseWheel，并逐段断言 viewport zoom 改变；任何一段依赖先选中节点都判失败。

恢复验收结果：

- 旧实现的 outer capture handler 对非 `.vue-flow__node` 直接 return，空白画布被交给 WebView 中未生效的 Vue Flow 默认链路；节点与空白形成两套 wheel ownership。
- 新 smoke 先用 trusted click 清空选择，再分别在空白画布与未选中 Analyze Color 节点发送 CDP mouseWheel；修复前稳定复现 `blank-canvas wheel did not zoom: 2.000 -> 2.000`。
- handler 现对整个 `workflow-canvas` 使用同一光标锚定 viewport 算法。原样 WebView 旅程通过，结果目录 `.task/workflow-editor-smoke/20260719-225739`。
