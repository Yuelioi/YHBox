---
slice: "04"
title: 原子宏模型与宏编辑器
status: completed
---

## Outcome / Question

把“简易录制”恢复为可编辑、可验证、可安全回放的宏系统。宏必须显式保存每个输入的按下、释放与等待，能够表达 W 与 D 的交叠时序；不得再使用 grouped keys + duration 丢失原子事件。

## Completion criterion

- 建立版本化 Macro asset 与 tagged union：KeyDown、KeyUp、MouseDown、MouseUp、Click、Scroll、Sleep。
- 宏验证器就地报告重复 Down、无对应 Down 的 Up、非法持续时间和结尾未释放输入。
- 编辑器支持录制、新建空白、插入、删除、复制、批量选择、拖拽排序、动作搜索和按键捕获。
- 选择动作时显示该时刻 held keys/buttons；延迟作为独立 Sleep 行，不藏在相邻动作属性中。
- 回放使用 held-input 状态机；完成、停止、取消、失败和应用退出都释放全部输入。
- 从资源库录制时只进入宏资源库；从编辑器资源工作台录制时，保存后立即插入或绑定独立“回放宏”节点。
- 宏和精准录制在入口、资源类型、列表筛选、编辑器、端口类型与回放节点上保持隔离。

## Blocked by

- Stage A 已完成并通过完整门禁与真实 WebView smoke。
- 可复用现有 hook、input codec 和安全释放能力，但不能复用损失原子时序的 grouped action 模型。

## Verification

Slice 内只运行继续开发所需的定向测试：

- Macro schema/codec round-trip 覆盖 W Down、D Down、W Up、D Up 与 Click 展开。
- 验证器覆盖重复 Down、孤立 Up、未释放输入和合法交叠输入。
- held-input 状态机覆盖正常完成及 stop/cancel/error/exit 清理。
- 编辑器行为测试覆盖增删复制、批量、排序、捕获与 held-state 投影。
- Stage B 完成 Slice 04 与 Slice 05 后，统一运行 `task check`、Windows 真机录制/回放 smoke 和视觉验收。

## Out of scope

- RawDelta、连续鼠标移动、拖拽轨迹、微秒时间线和 counts360；这些属于 Slice 05 精准录制。
- 从精准录制隐式转换宏。
- 把宏自动拆成 PressKeys/Delay 工作流节点。

## Result

已完成。Macro v1、原子动作验证与编辑、held-input 安全释放、资源服务和回放节点已实现；编辑器与资源库使用同一不可变资源边界。录制保存恢复批次见 Slice 10。
