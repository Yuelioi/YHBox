# M3a — Release、Installation 与 Readiness 核心

## Journey

一个已经由信任边界验证的 Workflow Release 到达本机后，应用立即创建独立 Workflow Installation，
不等待节点包、目标、凭据或授权齐全。同一 Release 可以安装多份；每份有自己的本机身份，缺失设置不会
阻止用户离开安装流程或继续查看。旧 `.yotta-workflow` 私有导入仍是 unverified Source，不能冒充 Release。

## Scope

- 本切片建立 verified Release projection、Installation lifecycle 和 Readiness Report 的正式领域边界及
  Content Catalog 持久化。
- 本切片不定义 Registry/M4 的签名验证格式，不把 unsigned bundle 升格为 Release，也暂不提供
  target/credential materialization UI。
- Readiness 一次返回全部 dependency、target、credential、manual consent 和 schedule consent blocker；
  consent 绑定精确 Release digest，生命周期与 readiness 独立。

## Implementation

- `workflowinstallation.Module` 只暴露 verified install、Installation 查询和 Readiness 查询；调用方必须提交
  已由外部信任边界产生的 `VerificationReceipt`，Source canonical bytes、source digest、Release digest 与
  Publisher Attestation digest 必须一致。
- Content Catalog schema 4 新增 immutable `workflow_releases` 与多实例 `workflow_installations`。Release
  首次写入和 Installation 创建处于同一事务；相同 Release 可复用，Release identity collision 或
  Installation identity collision 均 fail closed。
- Readiness 对精确 Node Package version/digest、Target Profile Definition binding、Credential Requirement
  binding、manual run consent 和 schedule consent 分别生成 blocker 与 repair action，并独立计算
  `RunAllowed`、`ScheduleAllowed`。active/archived 只属于 Installation lifecycle。
- 桌面与 CLI composition root 已注入 Catalog repository 并公开 Installation Module；现有 Source 编辑、
  Bundle import 与 Workflow-ID run 路径没有被暗中改写。

## Verification

- 领域测试覆盖同一 Release 创建两份 Installation、未配置仍完成安装、一次返回五类 blocker、
  schedule-only blocker、完整 ready、非 canonical Source 与缺失 Release identity。
- Catalog 测试覆盖 Release + Installation 原子提交、多实例读取、精确字节恢复、Release identity collision
  回滚；migration fault 测试更新为 schema 4 前的完整 schema 3 对象集。
- 定向 Go 测试已覆盖 `workflowinstallation`、Catalog、appbootstrap、desktopapp 与 CLI composition。
- `task check` 最终退出 0：121 个变更文件路由到合同、AI 8/8、Wails 17 服务/148 方法、
  38 个受影响 Go 包和前端 format/lint/typecheck/i18n、82 个测试文件/351 项测试。

## Status

Finished.
