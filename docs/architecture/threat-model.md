# Threat model

## 信任边界

- Container/package、MCP 参数、HTTP 响应、文件和脚本内容都视为不可信输入。
- 操作系统账号、已选择窗口/Android 设备和用户显式配置视为本地授权边界。
- 第三方 Go/Node/Rust 依赖属于供应链边界。

## 高风险能力与缓解

| Surface | Main risk | Current controls |
|---|---|---|
| Desktop process privilege | 整个 UI/runtime 以管理员运行会把任意前端、插件或解析缺陷放大成主机级权限 | 主进程固定以调用用户身份启动，不调用 `EnsureAdmin`/`runas`；需要高完整性目标时必须由显式 capability/provider 实现，否则 fail closed，不能给桌面壳 ambient admin |
| MCP command surface | 未授权整图覆盖、越权 capability、schema 放大 | 旧 HTTP/runtime tools 已删除；3.1 只提供 bounded catalog、分页 inspect、revision-CAS typed patch、compile 与无副作用 run preview，全部调用同一 Application。MCP transport 默认不装配、不监听；未来显式 transport 也不得拥有旁路执行器 |
| Script node | 任意代码、宿主逃逸和资源滥用 | Script 3.1 只在一次性隔离 worker 中接收规范化 JSON；没有节点/service registry、文件、网络或进程绑定。宿主 admission 还要求精确隔离 feature，超时由宿主终止整个 worker |
| HTTP egress | SSRF、DNS rebinding、重定向绕过、响应放大和秘密头泄露 | 只允许显式安装并授权的 exact Origin；workflow 只给相对路径与查询；禁代理/重定向/Cookie/凭据/自定义头；DNS 与拨号地址复验，默认拒绝本机/私网/特殊地址；响应有超时、字节与 UTF-8 上限，只暴露 status/body/content-type |
| Installed application lifecycle | 任意命令、shell/argument 注入、错误进程终止、受信应用被替换 | workflow 只选 exact installed slot；档案封存绝对 `.exe`、SHA-256 和固定逐项 argv；拒绝 shell/script host、PATH、raw command line、工作目录/env/PID 输入；每次调用重验摘要，终止只匹配 OS 文件身份；dangerous consent 与 journal 不记录路径/argv/PID。该能力明确不是目标应用 sandbox |
| File/package import | 路径穿越、zip bomb、恶意 graph | Node Package archive 只接受 canonical manifest 的精确 regular-file payload set；entry/压缩与展开 bytes 有上限，路径/case/symlink/size/SHA-256/CRC 全部重验。安装只接受精确 digest 的 local approval，并以 immutable generation + registry-last 赋予 authority；quarantine/rollback/reopen 继续验盘。其他导入器新增时同样必须加 entry/size/count 上限 |
| Fetch/network | SSRF、敏感内网访问、巨大响应 | 节点行为必须显式、超时/大小受限；未来远程分享前需要 URL policy |
| Input/capture | 错误 target、按键残留、隐私图像 | capability/target validation、context cancel、held-input release、窗口句柄重校验 |
| Logs/settings | 凭据或隐私泄漏 | 不记录 secret 值；用户数据写失败可观测；安全报告避免附真实数据 |
| Dependencies/history | 已知漏洞、许可漂移、历史凭据泄漏 | Go/Node/Rust Dependabot、govulncheck、cargo-deny、固定 lockfile、第三方 license artifact、Gitleaks 全历史扫描 |

## 非目标

Yotta 的 capability/admission/provider 边界用于约束 workflow authority；它不防御能够替换本地二进制、修改受信设置/数据目录或以同一 OS 用户注入进程的攻击者。Script worker 是缩小 guest 权限和故障域的隔离层，不是对恶意本地管理员的安全边界。当前 MCP protocol 不暴露 Run admission 工具；未来若增加执行工具，宿主 approval 只是防误操作闸门，不是本机身份认证系统。

## 复查触发

监听地址不再是 loopback、引入远程分享/云同步、开放脚本宿主 API、增加 package importer、启用浏览器 target 或处理凭据时，必须更新本模型并做专门安全评审。
