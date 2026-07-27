# V4-Q Compatibility 与无调用表面清理

## Goal

删除已证明无 production 调用的 source-compatible/test convenience 表面；对仍能读取已发布磁盘状态的
compatibility reader 建立持久化改写、最低支持版本和可验证退役路径，不把 durable compatibility
误当普通死代码。

## Status

Finished

## Completed

- 删除 `appbootstrap.Runtime.ReplaceAutomation`，原测试改走 production `PrepareAutomation/Commit`。
- 删除未调用的 `App.GetLogSink` 和 `ApplicationService` 未读取的 `*App` 字段。
- 删除 automation `TargetKind` alias，测试显式使用 `TargetKindDesktopWindow`。
- 删除 production `services.NewApp` 与 `App.Shutdown` convenience；services 测试使用 `_test.go`
  helper，跨包测试显式调用 `OpenApp/ShutdownContext`。
- Settings retired `workflowConsent` 在 checksum 验证和严格迁移后由 `OpenSettingsStore` 原子写回
  下一 generation；fixture 覆盖四类 installation，并证明重开不再触发 rewrite。
- Node Package registry v2 在 package generation、signature 与 trust 全部验证后原子写成 v3，
  package-scoped grants 不再只存在内存。
- Run v1 已确认只在 root layout 1→2 导入：SQLite Ledger 与 import marker 是 durable authority；
  marker 后即使旧 JSON 损坏也不会再次读取。
- Blob v1 已确认只在 root layout 2→3 迁移：对象验摘要、分片迁移、inventory 对账完成后才发布
  v2 marker；普通 Blob Store 不含 legacy fallback。
- Migration journal 的唯一 writer 强制输出 document v2；历史 committed v1 journal 仍是只读兼容
  边界，待 writer lease 下的历史迁移后才能删除 reader。
- `docs/compatibility.md` 记录 V4 从 3.1.0 直接升级的 ledger、五个 durable boundary 和逐项退役条件。

## Verification

- `go test ./internal/appbootstrap ./internal/services ./internal/desktopapp -count=1`
- `go test ./internal/automation/installed ./internal/noderuntime ./internal/appbootstrap ./internal/architecture -count=1`
- `go test ./internal/services -count=1`
- `go test ./internal/nodepackage ./internal/run ./internal/blob ./internal/storage/migrate -count=1`
- 中间切片不运行 `task check`。

## Durable compatibility decisions

- settings retired `workflowConsent` 与 Node Package registry v2 已有 durable rewrite；提高最低直接升级
  版本前保留 reader。
- Run v1 import 与 Blob v1 migration 分别跟 root layout 1、2 支持窗口一起退役。
- storage migration journal v1 尚有 committed history；在历史文件完成 writer-lease migration 前保留。

## Next

进入 [Go 清扫最终交付](v4-r-go-cleanup-delivery.md)：清理 smoke 工具和架构文档，然后只在整个
Go 清扫完成点运行一次最终门禁。
