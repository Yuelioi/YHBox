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

## Status

In progress. 剩余工作是 Target Profile materialization、secure credential binding 与对应设置界面。
