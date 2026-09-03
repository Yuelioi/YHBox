# Configuration generations and lifecycle

本指南用于修改 Settings installation、execution environment、Run owner 或桌面后台组件。先从当前代码确认
调用链；目录名相似不代表它拥有同一生命周期。

## Ownership map

| Concern | Current owner |
| --- | --- |
| settings DTO、validation、durable generation | `internal/services/settings.go`、`settings_store.go` |
| AI/Network/Application/Automation installation 构造 | 各 domain package 的 `Install`/draft；组合在 `internal/localruntime` |
| immutable execution environment 与热替换 | `internal/appbootstrap` |
| Run queue、snapshot、cancel 与执行 owner | `internal/application` |
| target generation / per-Run sessions | `internal/targetruntime`、`internal/automation/installed` |
| desktop window、hook、schedule、recording、tool lifecycle | `internal/desktopapp`、`internal/appruntime` |

`main.go` 只保留进程入口和嵌入资源。不要把 service construction、storage open 或后台 goroutine 放回入口。

## Changing settings-backed installations

1. 在 settings DTO 定义用户可持久的意图；secret、native handle 和运行期 session 不能进入 JSON/RPC。
2. 在 `Validate` 和 domain draft/seal 处拒绝不完整配置。Adapter-owned schema 应由 registry descriptor 暴露，
   UI 不维护平行枚举。
3. `services.App.AttachSettingsActivator` 安装的 `SettingsActivationPreparer` 必须完整构造新的 AI、Network、
   Application 和 Automation generation；prepare 失败时 `MutateSettings` 不保存候选，旧 settings/environment
   保持不变。
4. `MutateSettings` 先 durable save 并发布内存 settings，再调用 activation plan 的 `Commit` 整体替换 execution
   environment。commit 失败属于 settings 已提交错误，必须显式报告并保证下次启动可从 durable settings 重建，
   不能假装回滚。成功后新 Run 使用新代，已持有旧 lease 的 Run 继续完成并在 owner close 时释放。
5. 测试至少覆盖 prepare failure 不发布、commit 后新旧 Run 分代、shutdown 释放和重开后的 durable 值。

Network、Application、Automation 是 Configured Target：配置即授权，per-Run direct invocation。不要给它们
增加逐节点 consent/grant/TTL、重复全量 validation 或身份扫描。稳定字段校验在配置/installation 时做一次；
operation 只检查执行该操作真正需要的前置条件和 deadline。

AI、workspace file、Blob、Stream 与隔离 guest 是 capability/provider 资源，仍按 admission 和 credential
binding 工作。不能为了复用代码把两类 authority 合并。

用户配置的可选外部工具不得成为应用启动依赖。installation 阶段只 seal 持久配置并建立惰性 provider；CLI、
worker、设备或远端服务是否存在，应在用户真正调用对应能力时检查并投影成该任务的 Problem。测试必须覆盖
“配置仍存在但可选工具不在 PATH”时 `localruntime.Open` 仍成功。

## Lifecycle rules

- 创建顺序和关闭顺序必须相反；构造中途失败要关闭所有已经成功创建的 owner。
- goroutine、hook、server、worker、target session、held input 必须有单一 owner、cancel 和 wait/close。
- `Close` 应幂等；不要只发 cancel 不等待，也不要让 callback 在 owner 关闭后继续访问已释放 service。
- Run 的 success、failure、cancel 和 application teardown 共享资源释放路径。新增 early return 时检查 lease、
  target、worker 和输入状态是否仍会关闭。
- Wails presentation 只注册/转发 service 与 event；核心 runtime 必须仍可由 headless CLI 通过 `localruntime`
  打开。

## Verification

- 对改动 package 运行 focused test；并发、generation、close 或共享 state 运行对应 `go test -race`。
- settings/store 变化测试 failed publish、recovery、reopen 和 unknown/retired field 行为。
- Target 变化做 Source → compile → Run snapshot → adapter → journal 纵向测试；native adapter 再跑对应真实
  Windows/ADB/Browser smoke。
- 最后运行 `task check`；影响整个 composition、发布 payload 或跨平台边界时再按构建指南选择 full/smoke。
