# Settings snapshot 与 durability
SUMMARY: Settings 读路径只返回 deep snapshot；所有写入经单一 writer 执行 clone → mutate → validate → same-directory atomic save → publish → ordered side effects，禁止暴露或原地修改 live 指针。
READ WHEN: 修改 Settings schema、SettingsService、热键/窗口配置持久化、autostart/logger side effect；排查设置并发覆盖、重启回退或损坏 JSON
RECHECK WHEN: 改变 settings 文件位置/格式；新增 settings writer 或 commit side effect；调整 Windows/Unix replace 实现

---

## 权威事务

- `App.Settings()` 返回 deep clone，调用方修改不会触及 live state。
- `App.MutateSettings` 用 `settingsUpdateMu` 串行全部 writer，从当前快照克隆后执行 mutator 与 `Validate`。
- rename 前任一步失败：不发布内存、不执行 side effect，旧文件保持可读。
- replace 成功后才发布新的内存快照。Unix 随后 fsync 目录；Windows 使用 `MoveFileEx(REPLACE_EXISTING | WRITE_THROUGH)`。
- replace 已完成但目录 durability barrier 报错时，错误带 `Committed() bool` 标记；内存仍发布新代，避免“磁盘新、内存旧”。调用方应记录 durability warning，不能回滚外部状态。
- autostart、LogSink 文件配置等 commit side effect 在 writer mutex 内按提交顺序执行。`settings:changed` 事件只通知消费者重读最新快照，因此可在锁外发送。

## 热键一致性

HotkeyRegistry 先切换 native binding，再调用持久化 callback。callback 的 rename 前失败必须把 entry/native binding 回滚到旧值；rollback native 注册失败则 entry 保留为 failed 并向 UI 可见。已提交的 directory-sync warning 由 executable 记录并视为一致成功，不能把 registry 回滚到旧值。

## 禁止模式

- 返回 live `*Settings` 后仅靠注释要求“不修改”。
- 先 swap 内存再 `os.WriteFile`。
- 在 SettingsService 事务锁之外异步应用会覆盖系统状态的 side effect。
- 吞掉 window resize 等 best-effort 保存错误；至少写结构化 warning。
