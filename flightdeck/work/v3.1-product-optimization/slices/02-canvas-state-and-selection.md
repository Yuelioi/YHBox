---
slice: "02"
title: 状态通道与专业级画布选择
status: completed
---

## Outcome / Question

让节点在运行后仍能清楚选择和编辑，并恢复专业工作流画布必需的框选、追加选择、切换选择、删除和平移手势。选择、执行、调试和校验不得再竞争同一视觉通道。

## Completion criterion

- Selection / keyboard focus 只占用稳定的 primary outline 或 ring。
- Execution 使用节点头部状态、语义色短条与非阻塞运行轨迹，不覆盖选择态。
- Debug 使用 warning 当前节点标记，Validation 使用就地错误和节点徽标。
- 空白左拖框选，Shift 追加，Ctrl 切换，Esc 清空，Delete 删除，Space 或中键拖动平移。
- Vue Flow store 拥有瞬时选择，Source 只在拖拽结束持久化位置；选择变化不造成位置回跳。
- 现有复制、剪切、重复、删除、对齐、分布、自动布局和折叠子图命令继续可用。
- 提供可发现的画布手势帮助和运行轨迹清除入口。

## Blocked by

- Slice 01 已完成。
- Slice 03 依赖本 Slice 稳定选择与状态表达。

## Verification

Slice 内只运行快速、红绿明确的前端定向测试：

- 节点样式投影测试覆盖 selected + succeeded、debug + selected、validation error。
- Vue Flow 选择契约测试覆盖框选、追加、切换、清空和删除命令。
- 位置同步回归测试覆盖 selection 变化不覆盖内部实时坐标。
- 手势帮助与清除运行轨迹入口具备组件级行为测试。
- 不在本 Slice 单独运行 task check 或 Windows 真机 smoke；它们在阶段 A 完成后集中运行。

## Out of scope

- 子图内部边界投影与接口编辑。
- 调试器链路修复或下线判定。
- 宏、精准录制、复杂节点和颜色分析。
- 自由停靠工作区或新的运行时状态。

## Result

- 新增节点视觉状态投影：primary outline 只表示 selection；执行使用独立短条，debug 和 validation 使用独立标记。
- 显式建立 Vue Flow 手势契约：左拖框选、Shift 追加、Ctrl 切换、Esc 清空、Delete 删除、Space/中键平移。
- 运行轨迹只在正在观察的路径上动画，terminal 状态不再永久占用边框；EditorSession 可清除上次轨迹。
- 保留现有批量命令并增加画布帮助、运行轨迹清除和默认关闭的 minimap 开关。
- 定向状态、选择、边和会话回归测试通过；阶段末聚合门禁与真实 WebView 框选旅程通过。
