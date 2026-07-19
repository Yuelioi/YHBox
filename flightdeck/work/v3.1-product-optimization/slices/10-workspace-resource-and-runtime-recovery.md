---
slice: "10"
title: 工作区资源与运行态恢复
status: completed
---

## Outcome / Question

恢复录制资源从“开始录制”到“保存、管理、绑定、运行”的产品闭环，并把键鼠宏、精准录制和视觉模板作为工作流编辑器内的就地资源工作区。运行结果不得覆盖选择和编辑反馈，顶部操作必须有稳定层级。

## Completion criterion

- 编辑器与资源库启动的录制在 Stop 后都可靠打开对应 Macro 或 InputClip 保存工作台；pending 可恢复、可丢弃，不依赖某个窗口本地的临时 origin。
- 工作流编辑器提供就地资源面板，分开显示键鼠宏、精准录制和视觉模板，支持搜索、查看、新建/录制/截图、编辑和把稳定引用绑定到节点。
- 资源库保留完整管理页；编辑器面板与资源库消费同一 Asset/BlobRef/recording service，不建立第二份资源状态。
- 录制目标在开始任务时选择，不把默认目标选择器长期占据资源库顶栏；创建、录制和筛选按当前资源类型就近出现。
- 运行完成节点不使用绿色边框和“已完成”占用选择通道；完成后仍可点击、选中、拖动和编辑，失败/暂停/当前执行才获得高注意力状态。
- 工具栏只保留一个明确主操作；保存、新建子图和辅助动作使用次级样式；自动化目标标签不换行并在窄宽度下可理解地收缩。
- 为 pending 保存所有权、完成态选择和资源面板关键动作建立确定性的前端回归测试。

## Blocked by

- Slice 04、05 已有未提交实现和通过的静态门禁。
- UAC 真机反馈已证明当前用户旅程失败，不能沿用先前完成判断。

## Verification

Slice 内先建立并运行可捕获用户原症状的快速测试：

- pending 状态事件到达后，无论录制由编辑器还是资源库发起，当前可见工作区恰好打开一次保存工作台。
- completed 运行快照存在时，点击节点仍产生可辨识 selection 状态。
- 编辑器资源面板能搜索 Macro、InputClip、Template，并将稳定资源引用交给绑定流程。
- 工具栏在目标名较长和常用桌面宽度下不换行、主次层级唯一。

与 Slice 06 一起完成恢复阶段后，再统一执行 `task check`、production build、UAC Windows 录制/保存/重开/绑定/回放 smoke 和真实 WebView 视觉验收。

## Out of scope

- 改变 MacroAction 或 InputClip 的已批准领域语义。
- 任意停靠式 IDE 布局系统。
- 复杂类型 Authoring Surface 和颜色分析控件；属于 Slice 07、08。
- 调试状态机修复；属于 Slice 06。

## Result

已完成。录制 Stop payload 的空 `Steps`/`Tracks` 在 Go clone 中从非 nil 空 slice 退化为 nil，序列化成 `null`，前端 strict guard 因而丢弃 pending；后端改为稳定空数组，前端 normalization 同时容忍 null。编辑器新增宏、精准录制、视觉模板资源工作台，保存录制后插入/绑定对应节点。成功态回到中性，普通点击显式更新 selection，工具栏只保留运行主操作，目标选择不换行。增强 WebView smoke 覆盖工作台三类入口和无控制台错误，并发现、修复了“全部分类”空 Select value。`task webview:smoke`、`task check`、`task build` 通过；正式 UAC 应用启动与桌面截图通过。
