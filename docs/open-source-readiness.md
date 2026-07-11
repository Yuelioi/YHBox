# Open-source release readiness

## 当前结论

工程结构已具备公开协作的主要技术门禁，但许可证尚未满足“开源项目”目标。

当前 `LICENSE` 禁止商业使用、营利分发和 SaaS。此限制与 OSI 开源定义中的自由使用/再分发要求冲突，因此准确称谓是 **source-available**。

## 发布前维护者决策

维护者必须选择一种路径：

1. 保留当前限制，并在 GitHub 描述与 README 中始终使用 source-available；或
2. 选择 OSI 许可证（例如 Apache-2.0/MIT，或希望网络服务修改也公开时评估 AGPL-3.0），完成版权与贡献授权审查后替换 `LICENSE`。

这不是代码代理可以代替权利人完成的决定。第三方 Go 依赖由仓库内 `cmd/check-go-licenses` 以 Apache-2.0/BSD/ISC/MIT allowlist 阻断检查并生成 report artifact；Node/Rust 使用各自生态门禁。项目自身许可证仍由维护者决定。

## 已闭合的技术项

- canonical module/repository identity
- PR/push test、vet、staticcheck、coverage、race 与跨平台 compile CI
- Go/Node/Rust Dependabot；govulncheck、Go/Node/Rust license 与 Rust advisory gate
- 架构、贡献、安全、平台、兼容和 threat-model 文档
- 非可信 parser/graph/package/MCP fuzz seed corpus
- Gitleaks 全历史扫描（1606 commits）通过；两个测试 UUID 使用精确值 allowlist，不豁免整个文件

## 外部发布设置

首次公开前必须在 GitHub 仓库设置中启用 private vulnerability reporting，并验证 `security/advisories/new` 可用。此项不能由本地源码提交完成。
