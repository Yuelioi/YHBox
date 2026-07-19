---
slice: "17"
title: 计划 Modal 创作流
status: completed
---

## Outcome / Question

创建和编辑计划作为当前计划列表上的临时任务完成，保留搜索、统计与列表上下文，不再用整页替换制造导航割裂。

## Completion criterion

- “新建计划”和“编辑计划”打开共享 BaseModal，列表与筛选状态留在背景。
- Modal 标题、取消、关闭与保存语义一致；未通过校验时错误靠近字段，保存失败不关闭。
- ScheduleEditorPanel 不拥有页面级滚动或标题；只负责表单和动作。
- 保存后列表原地更新；取消不持久化 draft。

## Blocked by

- 现有 ScheduleStore createDraft/save 和 ScheduleEditorPanel。
- Slice 16 完成编辑器快捷交互后统一做 Stage I 视觉验收。

## Verification

- SchedulesView 组件测试覆盖 create/edit Modal、取消和保存。
- 既有 schedule WebView journey 更新为 Modal 内创建、绑定工作流、保存、重新编辑。
- 视觉检查宽屏与最小支持窗口下的高度、滚动和焦点恢复。

## Out of scope

- 修改计划后端 schema、触发器或执行队列。
- 把计划编辑拆成独立路由。
- 新增计划批量管理。

## Result

已完成。SchedulesView 始终保留指标、筛选与列表，创建/编辑 draft 通过共享 BaseModal 承载 ScheduleEditorPanel；关闭或取消仅丢弃本地 draft，保存成功后复用 store reload 原地更新，失败保留 Modal。管理页壳、ScheduleEditorPanel 和创作基础共 19 项定向测试、TypeScript 类型检查与 i18n 检查通过。
