# Threat model

## Trust boundaries

- Workflow、Node Package、MCP/AI patch、HTTP response、文件、脚本和进程输出都视为不可信输入。
- 操作系统账号、用户选择的窗口/设备/浏览器与 Settings 中的 installation 是本地授权边界，不是内容本身。
- 第三方 Go/Node/Rust dependency、worker 和 package 是供应链或进程隔离边界。

## Current controls

| Surface | Main risk | Current boundary |
| --- | --- | --- |
| Desktop privilege | 自动化宿主权限放大解析、前端或 native adapter 缺陷 | Windows production manifest 使用 `requireAdministrator`；dev host 使用独立 `asInvoker` manifest。管理员权限不绕过 secure desktop 或受保护进程 |
| Workflow/MCP/AI authoring | 越权整图覆盖、schema 放大、revision 丢失 | 只接受 strict Source/typed patch、bounded inspect/catalog、compile/preview 和 revision CAS；全部进入同一 Application |
| Script/Process/Wasm | 任意代码、宿主逃逸、资源滥用 | 独立 worker、bounded protocol、明确 host feature/capability、超时与 owner 终止；没有等价 confinement 时 fail closed |
| Node Package | 路径穿越、zip bomb、签名/trust rollback、namespace 劫持 | exact manifest file set、路径/size/digest/mode/signature 重验，registry-last authority，publisher ownership、revocation 与 quarantine |
| Network/Application Target | 私网访问、任意进程启动或终止 | 只执行用户配置的 origin/path/argv 和 timeout/size；这是高权限本地 Target，不伪装成 capability consent 或 identity pinning |
| Automation Target | 操作错误窗口/设备/浏览器、键鼠卡住 | Workflow 绑定配置 slot；adapter 做目标解析和 deadline；Run owner 在 cancel/failure/teardown 释放 target 与 held input |
| Storage/import | 路径穿越、损坏覆盖、WAL 不一致 | profile identity/single writer、strict schema、durable publish、SQLite integrity/online backup、migration journal 与 quarantine |
| Credentials/logs | secret 或隐私泄漏 | API key 在 OS secure store；settings/RPC/trace 不回传 secret；日志和 fixture 脱敏 |
| Dependencies/history | 漏洞、许可漂移、历史凭据 | lockfile、Dependabot、govulncheck、cargo-deny、license gate、Gitleaks 与 pinned Actions |

## Non-goals and review triggers

Configured Network、Application 和 Automation Target 使用当前进程权限直接执行，没有 consent/grant/TTL；
用户配置本身就是授权。Script/Process/Wasm 隔离用于缩小 guest authority 和故障域，不对恶意本地管理员提供
安全边界。

增加云同步/远程分享、网络监听、脚本宿主 API、新 package importer、凭据类型、低权限/双进程架构或新的
外部执行 Target 时，必须先更新本模型和相应纵向测试。
