# M5b — Installation staged update

## Journey

用户为一个未派生的 Workflow Installation 获取并验签新 Release 后，先查看 exact Release diff 与候选
Readiness。兼容的本机 target/credential binding 原样保留，新增定义只补发布者默认值；用户确认后才在
一个事务中切换 Installation。切换前的验证、预检或持久化失败都不改变当前 Release。

## Boundary

- trust boundary 继续产出 `ReleaseRecord`；`workflowinstallation` 深 Module 只接受 verified immutable
  candidate，并校验 publisher namespace 与 workflow identity 和当前 Release 一致。
- opaque `PreparedUpdate` 锁定 current Release、configuration generation、candidate bytes、确定性 diff、
  本机配置迁移结果与 candidate Readiness；调用方不能伪造待提交状态。
- 相同 target definition/credential kind 保留本机值；新增 target materialize defaults、新增 credential
  保持 unbound。删除或改变仍承载非默认本机值/绑定的 slot 产生 migration conflict，不允许提交。
- consent 锁定 exact Release，切换时总是清空；依赖、target、credential 与 consent blocker 可在确认前展示，
  但不阻止用户把 Installation 切换到一个可查看、可继续配置的 not-ready Release。
- repository 以 expected Release + expected configuration generation 执行单事务 CAS；同时插入 candidate
  Release projection、切换 Installation 和写入 configuration，不建立第二套 update store。

## Verification

- compatible local target settings/bindings 与 credential bindings 保持逐字节不变；新增项只使用 candidate
  defaults，removed/changed configured slot fail closed。
- wrong publisher/workflow、same Release、invalid candidate、stale Release/generation 与 repository fault
  均不改变 current Installation/configuration，也不留下不完整切换。
- candidate dependency/target/credential/consent blocker 在切换前可见；确认切换后既有 run/schedule consent
  不会沿用到新 Release。
- 深 Module、Catalog transaction、composition contract 与增量 `task check` 通过。

## Status

Finished.

## Evidence

- 深 Module 测试覆盖 compatible local value preservation、新增默认值、candidate blocker、exact consent reset、
  publisher/workflow/version identity 拒绝与 configured target/credential conflict。
- Catalog 测试证明 candidate Release insert、Installation release switch 与 configuration generation update
  同事务提交；stale generation 会回滚已执行的 switch 且不留下 candidate projection。
- `go test -race ./internal/workflowinstallation ./internal/storage/catalog` 与 `task check`（34 个受影响
  Go 包）均退出 0；本切片无 Wails/UI 变更，不触发 WebView smoke。
