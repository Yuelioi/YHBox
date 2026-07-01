# Index — debug-step-fixes

## State

已完成后端修复，待用户在真实编辑器/游戏场景 smoke。

## Next

- 人工 smoke：调试 Start 后逐步经过普通节点、禁用节点、Win32WindowTarget/窗口依赖节点、Loop count、Loop forever，并用 Ctrl+Shift+F9 强停正在 stepping/running 的调试 session。
- 若还有“依赖 window 卡住”的具体节点，记录节点 kind、当前 target/window 状态、是否有 Fail 出口，再继续追执行路径。

## Read now

- flightdeck/knowledge/nodes/debug-step-region-runner.md

## Read if

- flightdeck/knowledge/nodes/node-system-architecture.md — 如果要扩展更多 RegionRunner 的调试语义。
- flightdeck/knowledge/nodes/node-data-flow.md — 如果调试队列里的 exec-data/held output 表现异常。
- flightdeck/knowledge/input/node-timed-input-loses-backend-activate.md — 如果窗口/输入节点在调试中表现为第一次不生效。

## Progress

Done:
- 复现并修复禁用 Loop 使用真实 `Done` 出口时队列断掉的问题。
- 复现并修复 Loop 在 `StepOnce` 中一次跑完整个 body 的问题；调试路径现在把 Loop 展开为带 `LoopFrame` 的 body token，并处理 Break/Continue。
- 复现并修复 Ctrl+Shift+F9 全局强停绕过 debug manager 的问题；热键回调现在走统一 StopAll。

Verified:
- `go test ./internal/services/container/runtime -run "TestDebugStepOnceSkipsDisabledLoopWithCanonicalDonePin|TestDebugStepOnceEntersLoopBodyOneNodeAtATime" -count=1`
- `go test . -run "TestStopAllForHotkey" -count=1`
- `go test ./internal/services/container/runtime -count=1`
- `go test . -run "TestDebugManager|TestStopAllForHotkey|TestContainerRunnerAdapter|TestServiceDebug" -count=1`
- `go test ./internal/services/container -run "TestServiceDebug" -count=1`
- `pnpm --dir frontend test -- src/stores/execution.debug.test.ts`
- `go test ./...`

## Open questions

- `ForEach` 和 `Subgraph/CollapsedNode` 是否也要实现“单步进入 region 内部”；本轮只修用户明确撞到的 Loop。
- “依赖 window 卡住”是否还存在独立根因；当前已覆盖强停调试 session，仍需真实窗口节点 smoke。
