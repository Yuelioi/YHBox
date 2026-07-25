# M3b — Installation-local 配置

## Journey

用户安装 verified Workflow Release 后，可以逐步保存本机 target/credential logical binding，分别授予手动
运行与计划运行 consent，随时退出设置并在之后继续。配置不完整时 Installation 仍存在，Readiness 始终从
持久事实重新计算；前端或调用方不能自行声称节点包已安装或授权已授予。

## Scope

- target/credential 只保存本机 logical profile/installation ID，credential secret 继续留在 secure store。
- 配置以 generation CAS 更新，允许缺项但拒绝 Source 未声明的 slot；两类 consent 锁定精确 Release digest。
- verified Release 的手动运行与 Schedule 只接受 Installation-ID；本地 Source 编辑器继续保留 Source-ID
  authoring/debug 路径，两者最终进入同一 compiler/admission/Run Ledger runtime。

## Implementation

- Content Catalog schema 5 为每个 Installation 建立一对一 `workflow_installation_configurations`，安装事务
  同时创建空配置；从 schema 4 升级时为已有 Installation 回填 generation 1 空配置。
- `workflowinstallation.Module` 提供 binding replacement、精确 run/schedule consent 和配置查询；所有更新
  使用 expected generation，stale writer 不覆盖新事实。
- Readiness 不再接受调用方提供 target/credential/consent；它从 Installation repository 读取。Node Package
  readiness 由 appbootstrap 注入当前 verified、enabled、host-compatible runtime package projection。
- `RuntimePackage` 补充 Publisher Namespace，使 readiness 能校验 Source 声明的 namespace/package/version/
  manifest digest 全部精确匹配。
- `PrepareExecution` 从一次 Installation/Release/configuration/dependency 快照复用 Readiness，按 run 或
  schedule scope fail closed，并产出不可伪造、不可变的 exact Release Source 与 target/credential selection。
- appbootstrap 将 prepared execution 接入现有 Application compiler、admission、Program Cache、Run Ledger、
  provider 与 worker 路径；Workflow Wails service 暴露 Installation list/readiness、精确 consent 和手动启动。
- Schedule schema 2 只保存 `workflow-installation` target；schema 1 Source reference 保留 ID 但强制停用，
  等待用户显式修复。启用保存先检查 schedule readiness，daemon 触发时再次准备 exact execution。
- Schedule UI 从 Installation 列表选择目标；仅缺 schedule consent 时显示共享确认框并授予精确 Release
  consent，存在 dependency/target/credential blocker 时保持停用并显示阻断。
- Content Catalog schema 6 为配置增加 canonical Target Profile 投影；新安装在同一原子事务中从 exact
  Release Definition materialize initial defaults，schema 5 记录首次读取时以 generation CAS 惰性补齐。
- Target Profile 更新按 Release 内嵌的 exact JSON Schema 校验 settings，并同步 Target Installation
  logical binding；未知 definition、schema drift、stale generation 与非 canonical 持久字节均 fail closed。
- Readiness 从当前 Automation Target generation 读取可用性与授权，只接受 Target ID、kind、adapter 和
  profile version 全部精确匹配的本机安装；Settings 替换目标后会立即更新该 projection。
- Workflow Wails service 暴露 Installation settings/query/update。工作流列表显示 Installation readiness、
  手动运行 consent 与 Installation-ID 执行；设置 Modal 只列兼容目标，覆盖 loading/empty/error/失效绑定，
  credential 区只保存 logical binding，不读取或伪造 secret。
- AI Credential Installation 投影提供稳定 binding ID、kind、非敏感标签与 secure store availability；
  更新必须精确匹配 Requirement kind 且当前可用，Readiness 也按实时 availability fail closed。设置 Modal
  只列兼容 profile，保留失效引用用于解释状态，无可用候选时引导到 AI 设置。

## Verification

- 领域测试覆盖 binding generation CAS、stale update、run/schedule consent lineage 和配置后完整 ready。
- Catalog 测试覆盖安装时空配置、canonical binding bytes、更新后重开投影和 stale generation 拒绝。
- `workflowinstallation`、Catalog、Node Package 与 appbootstrap 定向测试通过。
- execution preparation 测试锁定单次配置快照、opaque bytes clone、正确 admission target slot 和 typed
  not-ready RPC envelope；Application integration 证明 run/schedule consent 分离且复用唯一 runtime。
- Schedule 测试覆盖默认停用、缺少 readiness authority、启用前拒绝、fire-time 拒绝和 schema 1 disarm 迁移。
- `task check` 退出 0：35 个受影响 Go 包、Wails 17 服务/152 方法和前端 82 文件/351 项测试全部通过。
- `task webview:smoke` 退出 0；隔离 Catalog 预置 verified Installation，计划创建/保存/重开始终持久
  Installation-ID，`20260726-045653` 的计划、恢复、编辑器与资源库截图已目检。
- Target Profile 领域/Catalog/service 测试覆盖独立 materialization、exact schema、unknown definition、
  canonical reopen、generation conflict、失效/未授权目标 blocker 与 Wails 设置往返。
- 最终 `task check` 退出 0：36 个受影响 Go 包、Wails 17 服务/154 方法/228 模型和前端
  83 文件/353 项测试全部通过。
- Windows WebView smoke 首轮在旧 Analyze Color CDP 旅程遇到瞬态 `Promise was collected`；确认进程退出后
  重跑 `20260726-054225` 退出 0，已安装工作流列表、设置 Modal、计划、编辑器与资源面均通过，新增截图已目检。
- Credential 领域测试覆盖 kind/ID/availability 精确匹配、stale generation、失效安全存储 blocker 与非敏感
  projection；service 测试覆盖候选投影和更新往返。最终 `task check` 退出 0：35 个 Go 包、Wails
  17 服务/155 方法/229 模型、前端 83 文件/353 项测试全部通过。
- WebView smoke 连续复现旧 quick-add 长异步 `Runtime.evaluate` 的 `Promise was collected` 后，将 runner
  拆为短同步动作和 Go 侧 DOM 轮询，并增加 fake-CDP 稳定回归测试；`20260726-060626` 完整旅程退出 0。

## Status

Finished.
