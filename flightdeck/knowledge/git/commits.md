# 本仓提交检查

通用 Git 写法无需在知识库重复；本仓只额外约束以下行为：

- message 使用英文 `type(scope): subject`，祈使句、小写开头、无句号；正文只在需要解释动机、取舍或 breaking change 时写。
- 一个 commit 对应一个可独立审查/回滚的逻辑单元；不要混入无关格式化、重命名或顺手清理。
- 不跳 hook、不 force-push、不 push 远端；当前仓库默认提交到 `main`，但提交前仍确认分支和用户是否授权提交。
- 探索中的 spec、plan、topic、knowledge 不因每次措辞变化而碎提交。只有用户明确要求提交/定稿，或文档随已落地代码构成同一交付单元时才提交。
- 不写 AI 署名、`Co-Authored-By` 或生成器宣传。
- 暂存前先看完整 `git status --short` 和 diff，只加入本次文件。出现 `MM`/`RM` 时说明 index 与工作树内容分裂，必须确认最终内容已暂存，不能只提交纯重命名。
- 多行 message 使用当前 shell 的安全原生方式；不要把 Bash heredoc 与 PowerShell here-string 混用。
