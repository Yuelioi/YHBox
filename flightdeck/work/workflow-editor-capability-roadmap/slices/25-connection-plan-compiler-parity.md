# Slice 25 — 连接计划 / Compiler parity

## Outcome / Question

TypeScript 的即时连接计划与 Go Compiler 是否对相同类型边界给出稳定一致的 direct/invalid 结论？

## Completion criterion

仓库存在一份固定、可读、带预算的共享 fixture；TypeScript 测试通过 sealed Projection 解释它，Go Compiler/type solver 测试通过权威 Catalog 解释它。任何一侧改变 exact、assignable、generic constraint、union、list 或 nominal digest 边界时，另一侧测试会立即失败。

## Blocked by

Slice 21、Slice 23。

## Verification

共享 fixture 的 Go 与 Vitest parity 测试；阶段末完整 `task check`。

## Out of scope

把 conversion 候选当成 Compiler 的隐式许可；Compiler 只判断直接可执行边，显式转换仍是图中真实节点。List 协变仍保持关闭。

## Result

In progress。

下一批：

- 定义不复制 Catalog 语义的最小 case schema，限制 case 数量和表达式深度。
- 覆盖 exact、Integer→Number、反向拒绝、trait generic、union 全成员/任一成员、List 不变性和 semantic digest mismatch。
- Go 侧直接执行 Compiler 使用的 type solver；TS 侧执行 Projection `typeMatch`。
- 固定 case ID 与期望 disposition，避免两套测试各自写期望后仍然同步漂移。
