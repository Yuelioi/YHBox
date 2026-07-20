# Slice 25 — 连接计划 / Compiler parity

## Outcome / Question

TypeScript 的即时连接计划与 Go Compiler 是否对相同类型边界给出稳定一致的 direct/invalid 结论？

## Completion criterion

仓库存在一份固定、可读、带预算的共享 fixture；TypeScript 测试通过 sealed Projection 解释它，Go Compiler/type solver 测试通过权威 Catalog 解释它。任何一侧改变 exact、assignable、generic constraint、union、list 或 nominal digest 边界时，另一侧测试会立即失败。

## Blocked by

Slice 21、Slice 23。

## Verification

共享 fixture 的 Go 与 Vitest parity 测试；2026-07-18 完整 `task check` 通过，包含 162 frontend tests、65.5% Go coverage、vet/staticcheck、167 Wails models、production build 与 bundle budget。

## Out of scope

把 conversion 候选当成 Compiler 的隐式许可；Compiler 只判断直接可执行边，显式转换仍是图中真实节点。List 协变仍保持关闭。

## Result

Completed。

- `internal/workflow/compiler/testdata/connection_plan_parity.json` 是唯一共享 case 集，使用符号类型名，由 Go Catalog 和 TS sealed Projection 分别解析，不复制 semantic digest。
- fixture 固定为 12 cases，并校验版本、数量预算、唯一 case ID 和 expectation 枚举。
- 覆盖 exact named type、Integer→Number、反向拒绝、trait generic 接受/拒绝、source/target union、List exact、List 不变性、List 泛型元素和 stale semantic digest。
- Go 测试直接执行 Compiler 使用的 `typeSolver`；TS 测试执行 `typeMatch`，共享同一 expected match。
- parity 首轮发现 Go Compiler 曾把具体 `List<Integer> → List<Number>` 当成可统一；现已修复为具体 List 走 Catalog 不变性，只有含类型变量时递归绑定。
- TS 同步区分 concrete List 与 generic List，且现有 `List<String> → List<String|Number>` 回归测试保持拒绝。
