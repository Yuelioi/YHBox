# Go code sweep audit

Date: 2026-07-27

## Baseline

- 659 个 Go 文件，共约 131,438 行；生产代码约 89,502 行，测试约 41,936 行。
- 生产代码最大的核心目录包括 `internal/nodes`、`internal/workflow/compiler`、
  `internal/noderuntime`、`internal/automation/installed`、`internal/services` 与
  `internal/application`。
- 最大文件中既有应审查的手写代码，也有应排除的生成物和工具：`plugin.pb.go` 是生成物；
  `cmd/workflow-editor-smoke/main.go` 是验收工具；文件大不自动等于 Module 浅。
- `internal/desktopapp` 的依赖扇出最高，但 composition root 的高扇出本身合理。问题在于它同时持有
  产品策略和生命周期决策，而不是单纯因为 import 多。

## Confirmed findings

### 1. 启动和执行环境装配有最高杠杆的重复决策

`internal/desktopapp/desktop.go` 的 `Run` 同时完成 storage migration/open、settings/log、
四类 installation、appbootstrap、settings 热更新、十余个 Wails service、hotkey、窗口、tray 与
shutdown。`cmd/yotta/main.go` 的 `buildRuntime` 又重复 storage/settings/installations、limits、
worker 路径和 appbootstrap 配置。

`internal/appbootstrap/bootstrap.go` 把同一组 installation 分别投影为：

- runtime provider map；
- admission Host Profile；
- builtin Policy 的 target map；
- automation 热替换时又重建上述结果。

这让 Provider ID、artifact、ABI、target kind、resource kind 与 credential binding 的一致性依赖多段
循环同步。`docs/architecture/node-engine.md` 仍保留“手工二次投影导致 capability 漏装”的旧说明，
也证明这一决策曾经发生漂移。

清扫方向：先封存一个具体 execution environment，让安装事实一次派生 Profile、Policy、Provider
和 generation lease；再让桌面与 CLI 共享具体的 local runtime open/close。不要为它预留多实现
interface。

### 2. `workflow/compiler` 是真实的边界错位，不只是大包

同一包同时拥有：

- Program artifact/open/validation；
- Source compiler 与 type solver；
- Adapter ABI（`Adapter`、`Invocation`、`AdapterResult`、action/status contract）；
- production Executor、scheduler、debug controller 与 run state；
- 仅由测试调用的纯数据 `Interpreter`。

`internal/noderuntime` 的几乎每个实现文件和 plugin host 都为了 Adapter ABI 依赖
`internal/workflow/compiler`。因此“compiler”这个 seam 已穿透到运行适配层。

清扫方向：先把仅测试 Interpreter 移到测试支持；随后让 Program、Adapter ABI、Compiler 和 Executor
拥有准确边界，但 production 始终只保留一个 Executor。`noderuntime.Installed` 本身是深 Module：
外部只提供 `Builtins + Dependencies`，内部隐藏大量节点 adapter，不应按节点家族拆成公开包。

产品层还暴露了“编译”按钮、Wails RPC、CLI command 和 MCP tool。Source 与 Program 虽然都是 JSON，
但前者是可编辑文档，后者是完成子图展开、类型/端口解析、实现锁定、执行排序和 capability plan
封存后的执行计划，因此内部 transformation 具有 compilation 语义。不过当前显式 `CompileSource`
只返回临时 Program；`StartRun` 仍会重新生成并持久化 Program。它对用户的真实价值是诊断和预检，
产品界面应改称“检查工作流”，而不是暗示生成二进制或可复用构建产物。

### 3. `Application` 是必要统一入口，但内部承担了过多独立状态机

`internal/application/application.go` 约 1,439 行、公开约 35 个方法，内部同时管理：

- Source create/patch/prepare/commit/import/delete/recovery；
- compile/preview；
- Run admission、provider generation lease、queue/worker/cancel；
- debug session；
- blob reference inventory；
- lifecycle 与 contract migration。

GUI、CLI 和 Schedule 共用它进入唯一 Run 路径是正确的，不能通过增加第二个 facade 或 runtime 解决。
问题是 Source transition、Run coordinator 和 Debug session 的决策与锁都集中在一个具体对象中。

清扫方向：保留 `Application` 作为稳定命令入口，在包内用具体深 Module 分担 Source transition 与
Run coordination；先用现有集成测试锁定外部语义，再为新状态机建立聚焦测试。

### 4. Wails Workflow service 混入了可复用 use case

`internal/services/workflow/service.go` 约 942 行。RPC DTO 转换属于 presentation adapter，但分页、
搜索、facet、批量 metadata、批量导出、引用检查与批量删除策略也只存在于 Wails service。CLI、AI
或后续入口若需要相同行为，只能复制规则或绕过它。

清扫方向：DTO、Wails-friendly error/result 留在 service；library query、batch command 和引用保护
进入具体 application use-case Module。不要把所有方法继续塞回 `Application` 单文件。

根 `internal/services` 还把 settings owner、log runtime、event emitter、AI RPC、target discovery
与 autostart 放在一个横向包内。它适合在核心链稳定后按领域收敛，不应作为第一刀。

### 5. 已有一批可证明删除的生产表面

当前静态调用证据表明：

- `workflow/compiler.Interpreter` 只被同包测试调用；
- `appbootstrap.Runtime.ReplaceAutomation` 只被测试调用，生产使用 prepare/commit；
- `services.NewApp` 与 `services.App.Shutdown` 只被测试调用；
- `services.App.GetLogSink` 没有调用；
- `services.ApplicationService` 保存了从未读取的 `*App`；
- `automation/installed.TargetKind` source-compatible alias 只在测试使用，生产已使用语义化 kind。

这些项目适合作为首批删除或转成 `_test.go` helper，但必须随新边界的测试一起处理，不能只追求删行数。

### 6. durable compatibility 不是普通死代码

以下路径仍能读取已发布磁盘状态，不能与 source alias 一起删除：

- settings 中 retired `workflowConsent` 字段的一次兼容读取；
- node package registry v2；
- Run v1 store import；
- Blob v1 layout migration；
- storage migration journal 的旧版本。

其中 settings 目前在 checksum 校验后内存剥离旧字段，但没有在成功读取后立即把当前格式写回，因此该
reader 没有自然退役点。清扫需要先建立 compatibility ledger、最低支持版本和持久化改写证据。

## Modules to preserve and deepen

- `internal/appruntime`：很小的 lifecycle interface 隐藏并发、回滚、逆序关闭和幂等，属于深 Module。
- `internal/noderuntime.Installed`：大量 adapter 实现被一个安装入口隐藏，属于深 Module。
- `internal/artifact`：高 fan-in 来自 canonical artifact/digest 事实源，不是应拆散的工具包。
- `internal/run` 的 Record/Store/Owner 分离与显式 storage migration：持久化和临时 authority 边界清楚。
- `main.go`：已经只保留进程入口与嵌入资源，符合仓库契约。

## Delivery order

1. Startup/environment：消除桌面、CLI、动态 automation replacement 的重复事实派生。
2. Runtime seam：移出测试 Interpreter，校正 Program/Adapter/Compiler/Executor 边界。
3. Application/use case：深化 Source、Run、Debug 与 library command Module。
4. Compatibility/deletion：删除无生产调用表面，为 durable readers 建立可退役迁移。
5. Tooling/docs：拆分 2,944 行 smoke command，刷新互相矛盾的架构说明和边界测试。

每一步都必须减少一个可点名的重复决策或 seam penetration；只移动文件、改名或新增 pass-through
interface 不计为完成。
