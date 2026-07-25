# M3b — Installation-local 配置

## Journey

用户安装 verified Workflow Release 后，可以逐步保存本机 target/credential logical binding，分别授予手动
运行与计划运行 consent，随时退出设置并在之后继续。配置不完整时 Installation 仍存在，Readiness 始终从
持久事实重新计算；前端或调用方不能自行声称节点包已安装或授权已授予。

## Scope

- target/credential 只保存本机 logical profile/installation ID，credential secret 继续留在 secure store。
- 配置以 generation CAS 更新，允许缺项但拒绝 Source 未声明的 slot；两类 consent 锁定精确 Release digest。
- 本切片先完成配置事实源及 Readiness 接入；把现有 Workflow-ID run/schedule 路由切换为
  Installation-ID 并 fail closed 是下一步。

## Implementation

- Content Catalog schema 5 为每个 Installation 建立一对一 `workflow_installation_configurations`，安装事务
  同时创建空配置；从 schema 4 升级时为已有 Installation 回填 generation 1 空配置。
- `workflowinstallation.Module` 提供 binding replacement、精确 run/schedule consent 和配置查询；所有更新
  使用 expected generation，stale writer 不覆盖新事实。
- Readiness 不再接受调用方提供 target/credential/consent；它从 Installation repository 读取。Node Package
  readiness 由 appbootstrap 注入当前 verified、enabled、host-compatible runtime package projection。
- `RuntimePackage` 补充 Publisher Namespace，使 readiness 能校验 Source 声明的 namespace/package/version/
  manifest digest 全部精确匹配。

## Verification

- 领域测试覆盖 binding generation CAS、stale update、run/schedule consent lineage 和配置后完整 ready。
- Catalog 测试覆盖安装时空配置、canonical binding bytes、更新后重开投影和 stale generation 拒绝。
- `workflowinstallation`、Catalog、Node Package 与 appbootstrap 定向测试通过。
- `task check` 退出 0：14 个变更文件路由到 plugin contract 与 36 个受影响 Go 包，全部通过。

## Status

In progress.
