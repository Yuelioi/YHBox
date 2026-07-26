# V4 清爽工作流核心

## Goal

把 Yotta 重构为上手即用的本地工作流工具：用户只需理解工作流、目标、计划、资源和运行记录；
保留现有编辑、录制、资源与执行能力，同时删除 Release/Installation/Consent 等未产生用户价值的产品与代码路径。

## Status

Finished

## Current

V4 已切换完成：产品与运行路径只保留 Workflow，Target 继续保存每个用户不同的应用路径和设备配置。
已删除 Release/Installation、更新/回退、离线安装包等约一万行代码，版本切到 4.0.0。旧节点契约会在
确认配置、绑定和拓扑可无损保留时自动升级。

## Next

无；后续产品体验优化在现有 V4 Workflow 模型上继续，不恢复 Release/Installation 平行路径。

## Progress

- 2026-07-26 建立 V4 Work；V3 M5f 以 `645e0bad` 留下可恢复基线。
- 2026-07-26 V4-A 完成：Workflow/Schedule/Run 统一为 Workflow ID，删除 Workflow
  Installation/Release runtime、catalog repository、离线包和前端更新/回退 UI。
- 2026-07-26 V4-B/C 收口：保留现有每用户 Target 设置与单一 Application runtime，Wails 方法由
  149 减至 138，没有新建 V4 平行框架。
- 2026-07-26 真实 profile 中的 `fishing-v2` 从 revision 0 无损迁移到 revision 1，CLI 编译通过。
- 2026-07-26 `task check`、`task build` 和 `task webview:smoke:full` 全部通过；Windows EXE 产品
  版本为 4.0.0，完整桌面旅程状态为 passed。
