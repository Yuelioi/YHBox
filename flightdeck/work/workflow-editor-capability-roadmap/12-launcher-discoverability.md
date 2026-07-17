---
slice: "12"
title: 悬浮窗入口与可发现性
status: completed
---

# Slice 12：恢复悬浮窗启动入口

## Outcome / Question

用户无需先知道或配置热键，就能从主产品界面打开悬浮窗；既有 launcher 独立窗口、配置和热键成为同一能力的三个入口。

## Completion criterion

- 主壳中提供稳定、可发现且不干扰主要工作流的“打开悬浮窗”入口，并在 launcher 设置页提供立即预览/打开动作。
- 前端调用现有 backend.tools.openLauncher；窗口已存在时聚焦而不重复创建，失败使用 Nuxt UI 错误反馈。
- 热键继续作为快捷方式，不是唯一入口；设置页清楚展示当前热键、冲突和禁用状态。
- 没有 launcher item 时仍可打开并展示可操作空状态，能直接进入配置。
- Windows 启动、最小化、主窗关闭策略与 AlwaysOnTop/frameless 行为一致，不留下孤儿窗口。
- 导航、命令入口与无障碍名称通过键盘可达；打开成功不 toast。

## Blocked by

Stage 3 批量验收；主壳入口位置需按现有信息架构确定。

## Verification

先做前端调用和窗口幂等定向测试；Stage 4 完成后统一运行 task check、task build、Windows WebView 主入口/设置入口/热键 smoke 与人工视觉检查。

## Out of scope

重写 launcher 内容模型、系统托盘常驻策略、跨平台完整支持、在工作流图中启动 launcher。

## Result

Completed。主窗口标题栏新增键盘可达的“打开悬浮启动器”，launcher 设置页新增立即打开与当前 system.launcher-toggle 状态/冲突/未绑定提示并直达快捷键配置。OpenLauncher 仍由后端幂等复用窗口，再次调用 Show+Focus；成功无 toast，失败走统一 Nuxt UI error。空 launcher 新增“立即配置”，通过 OpenLauncherSettings 显示并聚焦主窗，再向只存在于主壳的路由监听器发送 settings/launcher 导航，不重载或绕过编辑器离开保护。主窗关闭时既有 tools lifecycle 仍统一关闭 secondary slots。tools/desktopapp Go 测试、16 个前端测试、1562-key i18n 与 typecheck 通过；正式 Windows window/hotkey smoke 留到 Stage 4 门禁。
