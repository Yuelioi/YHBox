# M4d — Import coordinator 与本机 authority 边界

## Journey

同一份 Installation Plan 无论由在线 source 逐项提供，还是由 `.yotta-offline-pack` 提供，都先通过同一
coordinator 完整暂存全部精确 artifact。只有所有 size/raw SHA-256 都匹配才产生 Session；任一 artifact
缺失或变化都会清理暂存且不声称完整。创建离线包还要求每项由上游明确判定仍可获取且允许再分发。

## Boundary

- `installationimport` 只协调 transport completeness；Session 不导入 Workflow、不安装 Node Package，
  也不授予 publisher trust 或 Workflow execution consent。
- Node Package root cryptographic authority 仍属于 `nodepackage.TrustPolicy`。用户 trust 是 registry v3
  中精确的 `publisherKeyId + packageId` grant；v2 只为已经安装的精确包迁移 grant。
- coordinator 分别暴露 `GrantPublisherTrust` 与 `InstallPackage`。安装前重新验证 staged raw identity、
  package signature 和 Plan 中的 publisher/package/version/manifest identity；信任一个包不会信任同 key
  命名空间内的其他包，新 release 也不会自动安装。
- Workflow execution consent 继续只由 `workflowinstallation` 对 exact Release 和 run/schedule scope 持久化；
  transport、trust 和 package install API 均不能写入该记录。
- `.yotta-offline-pack` 的低层容器不携带 authority；面向完整离线交付的 writer 必须先通过每项
  redistribution check，缺失、下架或禁止再分发时不得发布目标文件。

## Verification

- 在线与离线 transport 产生同一 Plan digest 和相同 staged artifact bytes。
- missing source 清理整个 Session；staged bytes 被换写后每次消费都会 fail closed。
- trust、install、execution consent 三种状态互不隐式推进；signed identity drift 被拒绝。
- package trust grant 按 `publisherKeyId + packageId` 持久化，重开有效但不扩展到 sibling package；
  legacy v2 只 grandfather 已安装包。
- `go test ./internal/installationimport ./internal/offlinepack ./internal/nodepackage -count=1` 与增量
  `task check` 退出 0。

## Status

Finished.
