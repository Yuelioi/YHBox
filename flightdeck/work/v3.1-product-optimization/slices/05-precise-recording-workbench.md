---
slice: "05"
title: 精准录制工作台
status: completed
---

## Outcome / Question

把精准录制作为独立于 Macro 的原始输入轨迹产品。InputClip 必须保留微秒时间、Seq、绝对移动、RawDelta、拖拽、滚轮、录制分辨率、mouse mode 与 counts360；工作台只提供不破坏这些语义的查看和裁剪。

## Completion criterion

- 资源库与编辑器使用独立“精准录制”入口，不存在 simple/precise 模式下拉框。
- 精准录制继续保存版本化 InputClip，不进入 MacroAction，也不隐式转换宏。
- 停录预览和资源详情使用专用工作台，分轨显示键盘、鼠标按钮、绝对/相对轨迹与滚轮统计。
- 支持设置起点、终点并裁剪；裁剪后重新归零时间并保留同时间戳 Seq 顺序。
- 暂停区间不会作为虚假等待写入录制；元数据明确展示分辨率、mouse mode、counts360 和校准风险。
- 提供原始事件分页/虚拟化诊断视图，不把数千事件全部渲染到 DOM。
- 工作流继续使用独立“回放输入录制”节点与 InputClip 端口，不能绑定 Macro。
- 从编辑器资源工作台保存 InputClip 后立即插入或绑定“回放输入录制”节点；资源库录制仍只入库。

## Verification

Slice 内只运行继续开发所需的定向测试：

- InputClip 裁剪覆盖同时间戳 Seq、重新归零、首尾持有输入补齐/拒绝策略。
- 精准预览覆盖 RawDelta、绝对移动、拖拽、滚轮和交叠按键统计。
- 工作台组件覆盖分轨、裁剪范围、原始事件分页与校准提示。
- Stage B 完成后统一执行 `task check`、UAC Windows 真机录制/回放 smoke 和视觉验收。

## Out of scope

- 轨迹平滑、速度曲线、任意事件行编辑。
- 从 InputClip 自动提取 Macro。
- 把精准录制展开成大量工作流节点。

## Result

已完成。InputClip 保留微秒时序、Seq、绝对/相对轨迹、拖拽、滚轮、环境与校准元数据；专用工作台提供分轨、裁剪和分页诊断，不与 Macro 混用。
