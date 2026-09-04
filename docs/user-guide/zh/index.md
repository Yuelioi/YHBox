---
title: Yotta 用户指南
description: 从安装到构建、运行和排查自动化工作流
slug: yotta
order: 1
source_id: yotta-user-guide-home
---

# Yotta 用户指南

[English](../en/index.md)

Yotta 是一款可视化桌面自动化工具。你可以把找图、鼠标、键盘、窗口、网络、数据处理和 AI 等操作连接成
工作流，并通过运行按钮、快捷键、计划或悬浮启动器执行。

> 当前文档对应 `4.0.0-alpha.2`。Alpha 版本仍可能调整界面和工作流格式，重要工作流请定期导出备份。

## 从哪里开始

- [快速开始](getting-started/index.md)：安装 Yotta，配置目标并运行第一个工作流。
- [认识主界面](getting-started/interface.md)：工作流、资源库、计划和设置分别负责什么。
- [工作流编辑器](workflow-editor/index.md)：添加节点、连接端口、保存、运行和调试。
- [工作流管理与分享](workflows/index.md)：整理、复制、导入、导出和迁移工作流。
- [节点入门](nodes/index.md)：如何选择节点，以及信号、数据、错误和状态端口的区别。
- [自动化目标](automation/index.md)：连接 Windows 窗口、Android 设备或浏览器页面。
- [资源与录制](resources/index.md)：管理键鼠宏和精准输入轨迹。
- [悬浮启动器](launcher/index.md)：在目标程序上方快速启动或停止高频工作流。
- [设置](settings/index.md)：配置语言、快捷键、目标、输入校准、AI 和 MCP。
- [Run 与调试](runs/index.md)：理解运行状态、时间线、取消和断点调试。
- [计划](schedules/index.md)：按时间、间隔、快捷键或手动运行工作流。
- [快捷键](shortcuts/index.md)：管理全局、录制、启动器和工作流快捷键。
- [更新与备份](maintenance/index.md)：升级 Alpha 版本并保护本机数据。
- [故障排查](troubleshooting/index.md)：处理保存失败、目标未就绪、快捷键冲突和识别失败。

## 建议的学习顺序

第一次使用时，先完成“快速开始”，再阅读“工作流编辑器”和“节点入门”。需要操作游戏或其他程序时，继续
配置“自动化目标”；需要重复点击、拖拽或识别画面时，再学习“资源与录制”。

Yotta 不要求所有流程都从录制开始。简单操作通常直接组合节点更清楚；只有需要复用连续键鼠动作、精准轨迹
或连续输入轨迹时，才需要先创建资源。
