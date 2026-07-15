# Yotta

[![CI](https://github.com/yottaapp/yotta/actions/workflows/ci.yml/badge.svg)](https://github.com/yottaapp/yotta/actions/workflows/ci.yml)
[![GitHub release](https://img.shields.io/github/v/release/yottaapp/yotta)](https://github.com/yottaapp/yotta/releases)

Yotta 是一个可视化自动化工作流平台。它用类型化节点图编排截图、视觉检测、输入、窗口、Android ADB、脚本和定时任务；当前产品体验围绕《异环 / Neverness to Everness》自动化打磨，但后端按可扩展 automation target 与 capability 设计。

> 平台状态：Windows 是完整支持的平台；Linux/macOS 当前目标是可编译、平台中立核心可测试，GUI 为预览级。详见[平台支持矩阵](docs/platform-support.md)。

## 当前能力

- 可视化节点图、子图、变量、表达式与 JavaScript 节点
- Win32 后台截图和输入、Android ADB target、内部 Browser CDP adapter
- 模板匹配、颜色检测、图像与文件/JSON/HTTP 节点
- Workflow 3.1 revision、崩溃一致保存、调度、日志与 typed MCP authoring protocol（默认不启动 transport）
- 内置钓鱼、弹琴、音游、战斗等《异环》工作流

## 从源码开始

需要 Go 1.25.12+、Node 22.18+、pnpm 11.1.2、Wails v3 CLI 与 Task；完整 Windows 构建还需要 Rust。

```powershell
task check
```

Windows 开发与打包：

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.117
task dev
task build
```

## 项目文档

- [架构入口](docs/architecture/README.md)
- [贡献指南](CONTRIBUTING.md)
- [兼容与迁移策略](docs/compatibility.md)
- [安全策略](SECURITY.md)与[威胁模型](docs/architecture/threat-model.md)
- [发布就绪状态](docs/open-source-readiness.md)

## 许可状态

当前 [LICENSE](LICENSE) 禁止商业使用和营利分发，因此本仓库是 **source-available，而不是 OSI 定义的 open source**。发布者若要以真正开源项目定位发布，必须先选择并替换为明确的 OSI 许可证；详见[发布就绪差距](docs/open-source-readiness.md)。
