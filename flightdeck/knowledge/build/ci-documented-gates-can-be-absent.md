# ⚠ 文档列出的验证命令可能根本没有进入 CI
SUMMARY: Yotta 曾在 README/CONTRIBUTING/build checklist 声明 frontend test/typecheck/i18n/format 应绿，但 ci.yml 未调用这些命令，导致 format check 已红而 CI 仍可能绿；发布就绪必须按 workflow 的实际 step 审核。
READ WHEN: 评估 CI、发布就绪、required checks 或声称“全套门禁已绿”时；文档列了验证命令但 GitHub Actions 结果异常少时
RECHECK WHEN: `.github/workflows/ci.yml`、Taskfile 或 frontend scripts 改动后

---

不要从 README、CONTRIBUTING 或 checklist 推断某项检查已被 CI 执行。必须同时确认：

1. workflow 有实际 step 调用同一 task/script；
2. 命令是 check-only，不带 `--fix`；
3. 失败退出码未被 `continue-on-error`、管道或 shell 语义吞掉；
4. required check 覆盖 PR 和受保护分支；
5. 本地 clean checkout 运行同一入口得到相同结果。

2026-07-13 审查发现：Go test/vet/staticcheck 在 CI 中，frontend production build 也在 GUI compile job 中，但 Vitest、vue-tsc、i18n、format check 都没有 CI step；本地 `pnpm -C frontend format:check` 对 188 个文件失败。这是门禁缺失，不是“已知非阻塞 warning”。
