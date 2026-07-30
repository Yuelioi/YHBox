# Threat model

## 信任边界

- Workflow/Node Package、MCP 参数、HTTP 响应、文件和脚本内容都视为不可信输入。
- 操作系统账号、已选择窗口/Android 设备和用户显式配置视为本地授权边界。
- 第三方 Go/Node/Rust 依赖属于供应链边界。

## 运行边界

| Surface | Main risk | Current controls |
|---|---|---|
| Desktop process privilege | Windows production UI/runtime 以管理员运行会把前端、解析或 native adapter 缺陷放大成主机级权限 | 产品发布构建为桌面自动化选择 manifest `requireAdministrator`，不维护按需 runas/双权限 fallback；`task dev` 仅为让 Wails fork/exec 监督开发进程而使用独立 `asInvoker` manifest，不改变发布契约；Network、Application 与 Automation Target 使用当前进程权限直接调用配置，Script/Process/Wasm guest 仍使用独立隔离 |
| MCP command surface | 未授权整图覆盖、越权 capability、schema 放大 | 旧 HTTP/runtime tools 已删除；当前只提供 bounded catalog、分页 inspect、revision-CAS typed patch、compile 与无副作用 run preview，全部调用同一 Application。MCP transport 默认不装配、不监听；未来显式 transport 也不得拥有旁路执行器 |
| Script node | 任意代码、宿主逃逸和资源滥用 | Script node 只在一次性隔离 worker 中接收规范化 JSON；没有节点/service registry、文件、网络或进程绑定。宿主 admission 还要求精确隔离 feature，超时由宿主终止整个 worker |
| Network Target | 配置可访问本机、私网或远程服务 | 用户配置 base URL、响应大小与超时；运行时直接使用标准 HTTP client、环境代理和 redirect，不做 capability、consent、grant、DNS/IP 分类或私网阻止 |
| Application Target | 配置可以启动应用或终止匹配进程 | 用户配置绝对路径与 argv；启动继承完整环境，允许 host 支持的任意文件类型；运行时不做 executable hash/identity、capability、consent 或 grant |
| File/package import | 路径穿越、zip bomb、namespace 劫持、签名或 trust rollback | Node Package archive 只接受 canonical manifest 的精确 regular-file payload set；entry/压缩与展开 bytes 有上限，路径/case/symlink/size/SHA-256/CRC 全部重验。Store 安装还要求 Ed25519 envelope、已知 publisher key 和该 key 对 manifest exact namespace 的显式 ownership；monotonic trust policy 与 signature evidence 和 generation pointers 在同一 canonical registry-last commit 中持久化。未知/撤销 key、撤销或 quarantine manifest、policy rollback、受阻 generation 的 enable/rollback/reopen 均 fail closed |
| Automation Target | 配置可以操作窗口、Android 设备或远程浏览器 | 运行时按配置直接调用；不固定设备 product/model、WebSocket authority 或目标 identity；context cancel 和 held-input release 只负责正常生命周期清理 |
| Logs/settings | 凭据或隐私泄漏 | 不记录 secret 值；用户数据写失败可观测；安全报告避免附真实数据 |
| Dependencies/history | 已知漏洞、许可漂移、历史凭据泄漏 | Go/Node/Rust Dependabot、govulncheck、cargo-deny、固定 lockfile、第三方 license artifact、Gitleaks 全历史扫描 |

## 非目标

Yotta 的 capability/admission/provider 边界仍用于 AI、File、Blob、Stream 和隔离 guest 等非 Target 资源。Network、Application 与 Automation Target 明确不进入这条边界：它们是用户配置，由 Run 直接调用，没有 consent、grant、identity pinning 或 TTL。Script worker 是缩小 guest 权限和故障域的隔离层，不是对恶意本地管理员的安全边界。

## 复查触发

引入远程分享/云同步、开放脚本宿主 API、增加 package importer 或处理凭据时，必须更新本模型。
