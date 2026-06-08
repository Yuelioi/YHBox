---
name: vacuous-defensive-cleanup-test
description: defensive-clear code (set fields = "" / nil after panic) needs a test that actually populates those fields BEFORE the panic — testing with a panic that fires before assignment is vacuous (passes whether cleanup runs or not)
when_to_read: 写 panic recover + 清 partial state 的测试 / 加 defer recover() 后想验 cleanup 起效 / review panic hygiene 测试时
applies_to: [node-framework, engine, test-design, panic-recovery]
last_updated: 2026-05-26
status: active
---

# Defensive-cleanup test must exercise the populated-then-panic path

## 教训

写 `defer recover()` 块清 partial result 的 cleanup 代码 → 必须用 "操作中途 panic, **部分** result 已写" 的路径测; 用 "操作开头就 panic, **零** result 已写" 的路径测是 vacuous test — 它对 cleanup 在不在都通过.

## 为什么

deferred recover 跑前, 关键看: 被 recover 的代码块在 panic 前给 result 写了多少. 若 `outs, err := fn()` 里 `fn` 一开头就 panic, `outs` 永远是 zero, 后续把 outs 拆给 result.ExitName/OutputData 的赋值行都不跑. 此时 cleanup 块 `result.ExitName = ""; result.OutputData = nil` 清的是已是 zero 的字段 — 删了 cleanup 测试仍过.

真要测 cleanup, panic 必须在 **assignments 之后** 发生. 唯一现实 path: 操作 success 返回 → result.ExitName/OutputData 被赋值 → 进入下游 callback (Display) → callback panic → recover 触发 → cleanup 清掉刚写的 result.

## 怎么 apply

写 defensive cleanup test 前问:
1. 哪行 panic? 哪行 assign-to-defended-field?
2. assign 在 panic 之前吗?
3. assert 检查 field == 0 是 trivial 0 (assign 没跑) 还是 cleared 0 (assign 跑了又被清)?

trivial 0 = vacuous. 重写测试或换一个真触发 cleanup 路径的 panic 源.

**真测 cleanup 的 stash-and-restore 验证**: 临时把 cleanup 3 行注掉跑测试. 测试**应 FAIL** (assertions 看到未清的 partial 值). 如果通过 — 测试是 vacuous, 改测试.

## Case 1 — 2026-05-26 C4a runWithRecover

C4a `engine.go::runWithRecover` 在 deferred recover 内显式 `result.ExitName = ""; result.OutputData = nil; result.DisplayText = ""`. 初版测试 `TestRunWithRecover_PanicClearsPartialOutput` 用 `partialOutputPanicNode.Run` 调 `ctx.Out("Out").Set("X", 42)` 然后 panic — 但 `.Set` 不返 Outputs, `.Fire()` 才返. fn panic 时 outs == nil, 后续 `result.ExitName = name` (line 67-68) 永远不跑. cleanup 清的是 trivial 0.

Code-quality reviewer 抓到. Fix: 用 Display callback panic — Run 正常 return Outputs (via `.Fire()`) → ExitName + OutputData 被写 → Display callback panic → recover → cleanup 才有活干. 新测 `TestRunWithRecover_DisplayPanicClearsPartialResult` + stash-and-restore 验证 (注掉 cleanup 后测试 FAIL: `ExitName="Out"` / `OutputData=map[X:42]`).
