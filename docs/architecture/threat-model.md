# Threat model

## 信任边界

- Container/package、MCP 参数、HTTP 响应、文件和脚本内容都视为不可信输入。
- 操作系统账号、已选择窗口/Android 设备和用户显式配置视为本地授权边界。
- 第三方 Go/Node/Rust 依赖属于供应链边界。

## 高风险能力与缓解

| Surface | Main risk | Current controls |
|---|---|---|
| MCP command surface | 未授权输入、越权 capability、错误 target | 旧 HTTP server 已退出生产 composition；3.1 MCP 恢复时只能调用 Application/Program Snapshot 命令，并复用 admission、Grant、target 与 capability policy，不能拥有旁路执行器 |
| Script node | 任意代码和资源滥用 | goja 隔离 JavaScript 本身，但 Script 可调用所有 `ScriptBindable` 节点，包括文件、网络和进程节点；graph/script 作者必须被视为拥有这些工作流能力，执行受 context、节点校验与显式 service registry 约束 |
| File/package import | 路径穿越、zip bomb、恶意 graph | ID/path 校验、schema/lock validation、原子提交；导入器新增时必须加 entry/size/count 上限 |
| Fetch/network | SSRF、敏感内网访问、巨大响应 | 节点行为必须显式、超时/大小受限；未来远程分享前需要 URL policy |
| Input/capture | 错误 target、按键残留、隐私图像 | capability/target validation、context cancel、held-input release、窗口句柄重校验 |
| Logs/settings | 凭据或隐私泄漏 | 不记录 secret 值；用户数据写失败可观测；安全报告避免附真实数据 |
| Dependencies/history | 已知漏洞、许可漂移、历史凭据泄漏 | Go/Node/Rust Dependabot、govulncheck、cargo-deny、固定 lockfile、第三方 license artifact、Gitleaks 全历史扫描 |

## 非目标

Yotta 和 goja Script 都不是权限隔离边界。能编辑并运行 graph/script 的作者可以组合已授权节点执行文件、网络和进程操作；能修改本地二进制、数据目录或以同一 OS 用户执行代码的攻击者也已在信任边界内。未来 MCP 的交互确认只是防误操作闸门，不是针对恶意本地管理员的认证系统。

## 复查触发

监听地址不再是 loopback、引入远程分享/云同步、开放脚本宿主 API、增加 package importer、启用浏览器 target 或处理凭据时，必须更新本模型并做专门安全评审。
