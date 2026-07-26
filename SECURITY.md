# Security Policy

## Supported versions

项目尚未发布稳定版。安全修复只保证进入 `main` 和最新发布版本；旧的预发布版本可能不回补。

## Reporting a vulnerability

请不要公开提交包含利用细节、凭据或用户数据的 issue。仓库发布到 GitHub 前，维护者必须先启用并验证 [GitHub private vulnerability reporting](https://github.com/yottaapp/yotta/security/advisories/new)；在入口启用前仓库不满足公开发布条件。当前没有可安全写入仓库的备用私密邮箱，不能用公开 issue 代替。

报告应包括受影响版本/commit、平台、攻击前提、影响、最小复现和建议缓解措施。维护者应在 7 天内确认收到；修复发布前双方协调披露时间。

## Scope

重点范围包括：MCP armed 绕过、任意文件读写/路径穿越、脚本逃逸、网络监听暴露、容器 package 导入、依赖供应链、凭据或日志泄漏、输入设备无法释放。普通游戏规则变化、检测准确率和需要本机已授权操作的行为错误通常不是安全漏洞。

架构假设和现有缓解见[威胁模型](docs/architecture/threat-model.md)。
