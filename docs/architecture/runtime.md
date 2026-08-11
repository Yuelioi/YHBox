# Workflow runtime

## Source → Program → Run

1. Workflow Source 保存在 Content Catalog，由 `internal/workflowstore.SourceStore` 读取；
   `internal/workflow/authoring.Engine` 只接受 typed commands，`internal/application` 以 base revision CAS 提交。
2. `internal/workflow/compiler.Compiler` 使用 sealed Node Catalog（其中的 Node Contract 引用精确 Data Type）和
   config validator 检查图并生成 immutable、content-addressed Program。Authoring Projection 从同一 Catalog
   单独派生给 GUI/AI/MCP，不是 Compiler 输入。
3. `internal/workflowstore.ProgramStore` 把 Program 放进 `cache/programs`。它是有配额、可淘汰、可重建的
   derived cache，不是 Workflow Source 或备份 authority。
4. `internal/application.Application` 先持久化 Program，再取得 provider/configured target generation 并完成
   capability admission；只有 durable queued Run 创建成功才进入 worker queue。同一个 `compiler.Executor` 和
   scheduler 执行普通 Run 与 Debug Run。
5. Run 状态、NodeAttempt、AdapterAction、values 和 events 写入 Run Ledger。进程重启会中断遗留 RUNNING、
   取消未交付 QUEUED，不猜测或透明重放外部副作用。

GUI、headless CLI 与 Schedule 的执行只能调用 Application command；桌面与 CLI 可以是不同进程/runtime
实例，但不能分叉执行语义。MCP 和 AI authoring 的修改也必须进入正式 Source/typed patch 边界。
presentation 不得绕过它们直接拼 Source Store、Compiler、Executor 或第二个 queue/debug runtime。

## Node execution

- `internal/datatype/` 定义精确类型和表示；`internal/nodecontract/` 定义端口、执行、错误、状态、Target、
  capability 和 implementation ABI。
- `internal/nodes/` 显式组装内建定义，`internal/nodecatalog/` seal immutable snapshot，
  `internal/nodeauthoring/` 从同一 Catalog 派生前端创作投影。
- `internal/noderuntime/` 把 exact NodeRef/implementation lock 绑定到 adapter。adapter 只接收 typed input、
  trigger、窄资源/Target session、action recorder 和 state binding，不拥有 graph scheduler。
- data edge 使用精确 TypeExpression；exec 与 error 是独立 signal route。status event 不参与图路由，但会先作为
  `NodeStatus` fact 写入 Run Ledger，再投影到 Run/debug UI。
- BlobRef 可持久化；Stream、Resource 和 held-input handle 只在所属 Run lease 内有效。

当前节点清单不写进文档。运行 `task nodes`、`task nodes:catalog` 或 `task nodes:authoring` 从 sealed builtins
生成当前视图；tracked schema 和内建投影由 `task contracts:check` 检查。

## Provider 与 Target

Settings 生成不可变 installation/environment snapshot，新 Run 使用新代，已经排队或运行的 Run 保持原代：

- AI、Blob、Stream、workspace file 和隔离 Script/Process/Wasm 等资源由 capability、admission 和 provider
  约束；credential 只作为 binding 在调用时解析。
- HTTP origin、已安装应用和 Automation Target 是用户显式配置，由 `internal/targetruntime` 直接调用，
  不进入 consent/grant/TTL 模型。
- Automation 目前通过统一语义描述 Win32 desktop window、Android ADB 和 Browser CDP；Workflow 绑定稳定
  slot，不持久化临时 HWND、设备连接或浏览器 session。

配置热替换必须整体发布新 environment；不能只更新一半 provider/target 表。Configured Target 的配置本身
就是授权，新 Run 按 generation 直接调用；不要在每个 node 前再增加 consent/grant/重复 target 验证。
Target、resource、held input 和 worker 的释放由 Run owner 负责，cancel/failure/teardown 都必须走同一清理路径。

## Lifecycle

`internal/localruntime.Runtime` 按顺序打开 profile writer lease、SQLite Foundation、settings、installations、
Blob/Package/Script runtime 和 Workflow runtime；`Close` 逆序关闭 Application、settings/log delivery、SQLite
和 writer lease。`internal/desktopapp` 只增加窗口、hook、recording 和 Wails presentation 的生命周期。

取消通过 `context.Context` 传播。任何 goroutine、server、hook、worker 或 native input 状态都必须有明确
owner，启动失败可回滚，关闭幂等并等待退出。
