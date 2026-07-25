# M4a — Installation Plan 合同

## Journey

在线 Registry 或离线包准备安装一个 Workflow Release 时，先产出同一份 canonical Installation Plan。
该计划锁定 Workflow Release artifact 与全部精确 Node Package artifact；客户端下载哪种传输载体都不会
改变制品身份，也不会借计划授予本机 publisher trust、package install、target/credential binding 或
run/schedule consent。

## Boundary

- `installationplan` 只拥有 transport-neutral artifact identity contract，不下载、不验签、不安装。
- `workflowbundle` 继续验证 data-only Source/Blob archive；`nodepackage.Store` 继续独占 TrustPolicy、
  signature verification 与 durable generation installation。
- `workflowinstallation` 继续独占本机 Installation 配置和 Workflow Execution Consent。
- Plan 对 Workflow 锁定 publisher/workflow/version/release/source/artifact digest，对 Node Package
  锁定 publisher/package/version/manifest/artifact digest；package 集合必须与 canonical Source dependency
  一一精确匹配。

## Verification

- 相同语义、不同输入顺序产生相同 canonical bytes 与 digest。
- 缺失、额外、publisher/package/version/manifest drift 全部拒绝。
- unknown/local authority 字段、非 canonical JSON、错误 media type/size/digest 全部拒绝。
- 后续在线下载与 `.yotta-offline-pack` 必须复用本合同，不建立第二套制品身份或信任语义。
- `go test ./internal/installationplan -count=1` 与增量 `task check` 退出 0。

## Status

Finished.
