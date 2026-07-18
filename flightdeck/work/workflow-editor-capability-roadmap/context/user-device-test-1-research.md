# 用户真机测试 1：研究摘要

## Research Read

本研究在本地源码和真机数据确认根因之后进行，只回答可编辑宏、运行反馈、单步语义、target 默认值。来源限定为厂商/产品官方文档与官方源码；完整检索稿位于 `.task/user-device-test-1-research.md`。

## Source Matrix

- Razer、Corsair、Logitech 官方宏文档：录制通道/延迟/鼠标移动是 preset；事件可增删改、重排、编辑 delay，Down/Up 必须完整。
- VS Code/Visual Studio 官方 UI、调试文档和源码：Output/Debug Console 位于可隐藏 Panel；默认在 debug break 而非普通启动时打开；大列表只渲染 viewport。
- Unreal Trace/Insights/Blueprint Debugger：高频 trace 使用独立 store、channel、range/filter；breakpoint 在节点执行前暂停并保留最近执行 trace。
- Blender Areas/Workspaces：任务视图可放在可调整区域，但 Yotta 3.1 不必实现完整自由停靠。
- GitHub Actions、Ansible：上层 default 被更具体配置覆盖，规则简短可解释。

## Patterns

- 简易/精准是同一 InputClip schema 的两种 preset；精准 move samples 折叠为 MovePath。
- 录制先保存资产；“添加到画布”只属于 editor invocation context。
- Logs/Timeline/Debug 共用底层 run events，但各自查询和呈现。
- 普通 run 不开面板；failure → Logs，pause → Debug，Timeline 按需。
- store retention、cursor/range query、virtualized/aggregated render 三层边界。
- debug 明确 previous/executed 与 current/will-execute；Step 等待同 runID 的新 sequence。
- target 采用 workflow default + node override；compiler resolve 后 runtime 仍显式。

## Local Application

3.1 实现可折叠/tab 化底部工作台，不造完整 dock framework；深化 RecordingSession、EffectiveTargetResolver、RunEvent projection、WorkspaceRoot 四个 seam。模板 picker 直接补 selected state 与固定确认。

## Next Step

按 Slices 39–42 实现，阶段末真机旅程、`task check`、production build/native smoke 统一验收。
