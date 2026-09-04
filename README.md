<p align="center">
  <img src="build/appicon.svg" width="112" height="112" alt="Yotta 图标">
</p>

<h1 align="center">Yotta</h1>

<p align="center">
  本地优先的可视化自动化工具<br>
  用节点连接窗口、键鼠、图像、Android、浏览器、网络和 AI
</p>

<p align="center"><a href="README_EN.md">English</a></p>

[![CI](https://github.com/yuelioi/yotta/actions/workflows/ci.yml/badge.svg)](https://github.com/yuelioi/yotta/actions/workflows/ci.yml)
[![GitHub release](https://img.shields.io/github/v/release/yuelioi/yotta)](https://github.com/yuelioi/yotta/releases)

Yotta 可以把重复操作变成可视化工作流。添加节点、连接执行顺序、选择目标和资源，就可以运行、调试、复用
或分享自动化流程，不必把日常操作都写成脚本。

![Yotta 工作流编辑器](docs/user-guide/assets/workflow-editor.png)

## 可以做什么

- **可视化工作流**：支持分支、循环、子图、变量、注释、类型化连线和自动布局。
- **桌面自动化**：控制 Windows 窗口、鼠标和键盘，支持窗口截图。
- **输入录制**：创建可编辑键鼠宏，或保留拖拽、连续移动和视角转动的精准轨迹。
- **更多目标**：连接 Android ADB 设备、Chrome/Edge 页面、HTTP 服务和本机应用。
- **运行与调试**：通过时间线、错误定位、断点和单步查看工作流行为。
- **高频与定时操作**：使用悬浮启动器、快捷键和计划启动或停止工作流。
- **AI 辅助**：调用本机配置的模型，或生成需要你确认的工作流修改建议。

## 工作流与编辑器

工作流首页支持搜索、分类、标签、排序、导入和导出。进入编辑器后，可以在同一画布上完成节点添加、连线、
配置、检查、运行和调试。

![工作流首页](docs/user-guide/assets/workflows.png)

工作流只保存可移植的逻辑和目标槽位。应用路径、窗口匹配、设备地址和凭据保留在本机设置中；从其他电脑
导入工作流后，只需重新绑定本机目标。

## 输入录制

键鼠宏适合逐条编辑按键、点击、滚轮和等待；精准录制适合连续移动、拖拽和相对鼠标视角。

<table>
  <tr>
    <td><img src="docs/user-guide/assets/macro-editor.png" alt="键鼠宏编辑器"></td>
    <td><img src="docs/user-guide/assets/precise-recording.png" alt="精准录制编辑器"></td>
  </tr>
  <tr>
    <td align="center">键鼠宏</td>
    <td align="center">精准录制</td>
  </tr>
</table>

## 下载与安装

当前版本为 **4.0.0-alpha.2**，主要支持 **Windows 11 x64**。

1. 前往 [GitHub Releases](https://github.com/yuelioi/yotta/releases) 下载 Windows 发布包。
2. 解压到固定目录。
3. 运行 `Yotta.exe`。

> Alpha 版本仍可能调整界面和工作流格式。重要工作流请定期导出备份。Linux 和 macOS 目前为预览级。

## 第一次自动化

推荐从一次安全的桌面点击开始：添加桌面应用、创建 Windows 自动化目标、新建工作流、选择工作流默认目标，
再连接“Run 开始”和“点击指针”。完整九步教程见[快速开始](docs/user-guide/zh/getting-started/index.md)。

![Yotta 设置](docs/user-guide/assets/settings.png)

## 用户文档

- [快速开始](docs/user-guide/zh/getting-started/index.md)
- [认识主界面](docs/user-guide/zh/getting-started/interface.md)
- [工作流编辑器](docs/user-guide/zh/workflow-editor/index.md)
- [节点入门](docs/user-guide/zh/nodes/index.md)
- [自动化目标](docs/user-guide/zh/automation/index.md)
- [资源与录制](docs/user-guide/zh/resources/index.md)
- [悬浮启动器](docs/user-guide/zh/launcher/index.md)
- [设置](docs/user-guide/zh/settings/index.md)
- [Run 与调试](docs/user-guide/zh/runs/index.md)
- [计划](docs/user-guide/zh/schedules/index.md)
- [快捷键](docs/user-guide/zh/shortcuts/index.md)
- [更新与备份](docs/user-guide/zh/maintenance/index.md)
- [故障排查](docs/user-guide/zh/troubleshooting/index.md)

## 数据、隐私与许可证

工作流、设置、资源和运行记录保存在本机。只有你主动配置并运行对应节点时，Yotta 才会访问网络服务或 AI
提供商。请只配置你信任的程序、设备、网页和接口地址。

Yotta 当前采用 [source-available 许可证](LICENSE)，允许个人、教育和研究用途，但不是 OSI 定义的开源软件。
商业使用、营利分发、SaaS 或付费服务需要另行授权。

## 开发与贡献

- [贡献指南](CONTRIBUTING.md)
- [安全策略](SECURITY.md)
- [开发者知识入口](docs/README.md)
- [架构与代码地图](docs/architecture/README.md)
- [发布流程](RELEASING.md)

从源码运行时，以 [`go.mod`](go.mod)、[`frontend/package.json`](frontend/package.json) 和
[`Taskfile.yml`](Taskfile.yml) 为工具链与命令真值。常用入口为 `task dev`、`task check` 和 `task build`。
安装仓库固定的 Wails CLI：

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.6
```
