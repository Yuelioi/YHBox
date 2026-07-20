# ⚠ 文档列出的验证命令可能根本没有进入 CI
不要从 README、CONTRIBUTING 或 checklist 推断某项检查已被 CI 执行。必须同时确认：

1. workflow 有实际 step 调用同一 task/script；
2. 命令是 check-only，不带 `--fix`；
3. 失败退出码未被 `continue-on-error`、管道或 shell 语义吞掉；
4. required check 覆盖 PR 和受保护分支；
5. 本地 clean checkout 运行同一入口得到相同结果。
6. 显式 `go test`/race/portable package 列表可由 `go list` 解析；目录仍存在不代表其中还有 Go package。

2026-07-13 审查发现：Go test/vet/staticcheck 在 CI 中，frontend production build 也在 GUI compile job 中，但 Vitest、vue-tsc、i18n、format check 都没有 CI step；本地 `pnpm -C frontend format:check` 对 188 个文件失败。这是门禁缺失，不是“已知非阻塞 warning”。

2026-07-16 复查：`quality-windows` 已直接运行 canonical `task check`，上述 frontend 缺口已关闭。Wave D 删除旧 Node/Container packages 后，race/portable-core 的独立列表曾继续引用已删除 package，以及只剩空目录的 `internal/services/inputclip/runtime`；必须实际执行列表或用 `go list` 验证，不能用 `Test-Path` 判断 package 存在。
