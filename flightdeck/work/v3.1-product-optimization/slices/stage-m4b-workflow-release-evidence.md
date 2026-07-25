# M4b — Workflow Release evidence container

## Journey

用户导出私有 `.yotta-workflow` 时，文件明确保持 `unverified`，导入只创建可编辑 Source 副本。Registry
准备发布制品时，可以把 Publisher Attestation 与 Platform Publication Proof 的原始字节附在同一
data-only archive 中；桌面端先验证 manifest 锁定的 path/media type/digest/size，但不会因“文件存在”
就把它们当成可信证明、安装代码或授予运行权限。

## Boundary

- Workflow bundle manifest version 2 显式保存 `sourceTrust: unverified` 与 uniquely-sorted evidence refs。
- evidence 只接受 `publisher-attestation` 和 `platform-publication-proof` 两种固定 path；payload 保持
  opaque，避免桌面端复制 Identity 的 in-toto/JCS/DSSE 合同或预定义尚属于 Registry 的 proof schema。
- `workflowbundle` 只做 ZIP 安全、字节预算和 exact digest 验证。未来 trust adapter 验证 proof 后才能
  产生 `workflowinstallation.VerificationReceipt`；普通 Import 始终只是 Source authoring copy。
- version 1 unsigned bundle 通过显式 reader 迁移为 `unverified` 观察结果；version 2 writer 不再省略该状态。

## Verification

- 不同 evidence 输入顺序产生完全相同 archive bytes；Inspect 返回固定顺序和 `unverified`。
- evidence 缺失、篡改、重复/未知 kind、错误 path/media type/size/digest 与 undeclared entry 全部拒绝。
- Import/Replace 确认文案明确不授予 publisher trust、不安装 Node Package、不授权执行，并单独报告待验证
  evidence 数量。
- `go test ./internal/workflowbundle ./internal/services/workflow`、前端 typecheck/lint/i18n/test 与增量
  `task check` 通过；门禁覆盖 10 个 Go 包、Wails 17/155/229 和前端 83 文件/353 项测试。
- Windows WebView smoke `20260726-062735` 完整旅程退出 0。

## Status

Finished.
