# R7 — 正式 data-root migration 与恢复 UI

## Outcome

把已发布 storage layout 1 升级为 layout 2。迁移在任何 domain Store 打开前完成 dry-run、空间估算、
自动 snapshot、journaled apply、双库 verify 与 root manifest commit；任一持久化边界中断后可继续或
回滚，不删除旧 authority。

## Current

Finished。核心迁移 Module、CLI 与独立启动期恢复窗口已经落地；旧布局 fixture、layout 1 → 2
immutable registry/checksum、快照、resume/rollback、单条 legacy Run quarantine/restore、诊断导出和
kill-point/tamper tests 已通过。

旧 JSON 和 snapshot 清理由未来显式 retention 动作处理，不属于本切片自动提交路径。

## Deep module

- `internal/storage/migrate` 是唯一跨 storage Profile、Catalog Foundation、legacy Run import、snapshot、
  journal、rollback 与 root manifest publication 的 Module。
- `storage.OpenForMigration` 只提供旧 layout 的正常 writer lease；`PublishCurrentLayout` 是唯一 commit point。
- `run.Store` 构造不再触发迁移。legacy JSON import 是 root migration 的显式 action。
- GUI、CLI 和自动启动迁移都调用同一 Interface，不复制 schema 或文件操作。

## Lifecycle

1. Inspect 只读 root identity、固定 authority 集、legacy Run 数量/字节、snapshot 估算和磁盘可用空间。
2. Apply 获取旧 layout writer lease，固化 plan 和 immutable step checksum。
3. checkpoint 已关闭的 SQLite authority，生成带每文件 SHA-256 的完整 snapshot manifest。
4. journal 进入 prepared/applying/verifying；Catalog migration 与 legacy import 可重复执行。
5. 两库 quick-check/schema health 通过后发布 layout 2 root manifest，再提交 journal。
6. current manifest 与未提交 journal 的 crash window 由下一次 Ensure 验证 snapshot/Catalog 后协调为 committed。

journal 与 snapshot 都按不可信输入处理：byte budget、strict JSON、step identity/checksum、状态、时间、
固定 authority 集、路径、大小与 digest 任一不符都 fail closed。

## Recovery surface

- CLI：`migrate plan|apply|resume|rollback|list|quarantine|restore|export`。
- GUI：主应用数据库打开前启动独立 Wails recovery window，展示 layout、空间、backup、legacy Run 与日志状态。
- 恢复窗口支持继续、回滚、隔离/恢复阻塞记录和脱敏诊断导出；commit 后退出恢复进程并重启 Yotta。
- 隔离只接受 journal 指明的一个 JSON basename，保留原始字节、大小和 SHA-256；restore 是显式逆操作。

## Verification

- [x] 仓库内冻结 released layout 1 byte fixture。
- [x] Inspect read-only、空间不足/未知 root entry preflight block。
- [x] prepared、Catalog apply 前后、manifest commit 前后 kill-point 可恢复。
- [x] snapshot rollback 恢复 layout 1 authority，随后可重新 Apply。
- [x] journal checksum 与 snapshot authority-set 篡改 fail closed。
- [x] invalid legacy Run 可隔离、恢复、再次阻断、再隔离并完成 resume。
- [x] CLI plan 保持只读，apply 提交 layout 2。
- [x] GUI recovery handler 同源 mutation guard、resume/restart contract 与静态 UI 检查。
- [x] `task check` 增量门禁。
- [x] `task build`、production CLI layout 1 → 2 upgrade/health。
- [x] Windows recovery GUI 与真实进程 kill/relaunch smoke。

## Evidence

- targeted race：`internal/storage/migrate`、`internal/storage/catalog`、`internal/run` 全部退出 0。
- 最终 `task check` 同一可续接进程退出 0：check router self-test、AI eval 8/8、bindings contract、
  32 个受影响 Go 包 test/vet 通过。
- `task build` 退出 0：3.1.0 Windows GUI metadata 与隔离 desktop startup smoke 通过。
- `task smoke:storage-migration` 使用 production 二进制退出 0：plan 未写状态，invalid legacy Run
  进入 recovery-required，recovery GUI 强停后 journal 保持，quarantine/resume 提交 layout 2，
  Content schema 3 与 Run schema 2 healthy，迁移后 GUI 存活。
- 会话内浏览器无可用实例，因此未附加浏览器截图；GUI handler、production process、响应式静态资源和
  可访问性约束由自动测试覆盖。

## Non-goals

- 不在迁移 commit 后自动删除旧 Run JSON、snapshot 或 quarantine。
- 不允许普通 Store constructor 迁移、降级或清理用户数据。
- 不把 source-available 产品描述为 OSI open source。
- 不以 Windows cross-compile 替代 Linux/macOS 原生 CI。
