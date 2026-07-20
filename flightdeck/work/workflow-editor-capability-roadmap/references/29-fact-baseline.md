# Slice 29：3.1 恢复事实基线与发布账本

## Outcome / Question

固定 3.0 行为 oracle、3.1 capability ledger、黄金旅程、既有 dirty-worktree 归属和历史 Knowledge 退役清单，让后续实现只围绕同一套发布事实推进。

## Completion criterion

- detached reference worktree 固定在用户指定 commit，且不接收实现改动。
- capability ledger 覆盖编辑器、自动化、录制、资产、节点、工作流管理和平台能力，并给出恢复/替代/删除决策。
- 黄金旅程覆盖 Windows、Android、Browser、录制、资产规模、类型、调试和错误恢复。
- 主工作树既有改动完成路径级归属，不被误认为新阶段证据。
- 3.0 历史 Knowledge 有完整退役 registry 与 R5 清理门禁。
- Slice 27 已拆成独立、可验收、可提交的执行 Slices。

## Blocked by

无。

## Verification

- `git -C E:\projects\organizations\yottaapp\yotta-3.0-reference rev-parse HEAD` 精确等于 `8316d590dbc8429b783b99982ff30d15e650c59a`，且 status 为空。
- capability ledger 每个 P0/P1 能力均路由到一个 owner Slice 和至少一个 golden journey。
- retirement registry 中的每个路径当前都带历史 marker；R5 action 非空。
- 当时的 Topic local links、frontmatter、`git diff --check` 和工作台完整性检查通过。

## Out of scope

- 不改产品行为。
- 不把旧 Container 代码迁回主树。
- 不为未发布 schema 增加数据兼容层。
- 不在 R0 运行 `task check`、production build 或 native smoke。

## Result

Completed。

- reference worktree 固定到 `8316d590…` 且干净。
- capability ledger 共 57 个能力项，全部有 3.1 决策、owner 和旅程/gate。
- G01–G17 覆盖核心创作、Windows、录制、资产、错误、workspace、Android 和 Browser。
- 18 条 3.0 Knowledge 已进入 R5 退役 registry。
- Slices 29–37 均具备独立 outcome、completion、blocker、verification 和 out-of-scope。
- `git diff --check`、18 个计划文档本地链接、Slice 结构和 retirement marker 检查通过；R0 不运行产品构建门禁。
