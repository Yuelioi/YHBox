# Public-release readiness

当前仓库具备公开协作所需的主要技术门禁，但 `LICENSE` 禁止商业使用、营利分发和 SaaS，因此准确称谓是
**source-available**，不是 OSI 定义的 open source。

维护者必须在公开定位前选择：

1. 保留当前限制，并在 README、仓库描述和发布材料中持续使用 source-available；或
2. 完成版权与贡献授权审查后，选择并替换为明确的 OSI 许可证。

这项法律/权利决定不能由代码代理代替。依赖许可、漏洞和供应链由 Go/Node/Rust 门禁、固定 lockfile、
pinned Actions、Dependabot、CodeQL/Gitleaks 等技术检查覆盖；它们不能改变项目自身许可。

首次公开发布前还必须在 GitHub 启用并验证 private vulnerability reporting，并完成证书、canonical
identity、维护者/owner、真实 Windows/Target smoke 与发布候选流程。当前安全报告入口见
[SECURITY.md](../SECURITY.md)，平台边界见[平台支持](platform-support.md)。
