# Index — target-controller-upgrade

## State

破坏性大升级 topic。调研、总体设计、Phase 1 抽象层、Phase 2 controller-call trace foundation、Phase 3 runtime trace ownership、Phase 4 keyboard controller routing、Phase 5 click controller routing、Phase 6 move controller routing、Phase 7 scroll controller routing、Phase 8 trace source metadata、Phase 9 text controller routing、Phase 10 mouse hold/drag controller routing 已完成并提交。核心决策：Go 保持主运行时，Rust 只作为 Win32/native controller hot path；先引入 `Target / Controller / CoordinateSpace / Trace`，再迁移节点、Android、浏览器和输入后端矩阵。

## Next

Plan Phase 11: decide whether `MouseMoveRel` should become a controller action. It is currently the only remaining runtime `InputService` method that still bypasses `Win32Controller`; relative camera/game movement may need a separate raw-input/controller policy instead of normal pointer coordinate semantics.

## Read now

- design.md
- plan.md
- plans/phase2-trace.md
- plans/phase3-runtime-trace.md
- plans/phase4-keyboard-controller.md
- plans/phase5-click-controller.md
- plans/phase6-move-controller.md
- plans/phase7-scroll-controller.md
- plans/phase8-trace-source.md
- plans/phase9-type-text-controller.md
- plans/phase10-mouse-hold-drag-controller.md
- ../../knowledge/architecture/target-controller-phase3-notes.md
- ../../knowledge/architecture/target-controller-phase4-notes.md
- ../../knowledge/architecture/target-controller-phase5-notes.md
- ../../knowledge/architecture/target-controller-phase6-notes.md
- ../../knowledge/architecture/target-controller-phase7-notes.md
- ../../knowledge/architecture/target-controller-phase8-notes.md
- ../../knowledge/architecture/target-controller-phase9-notes.md
- ../../knowledge/architecture/target-controller-phase10-notes.md

## Read if

- ../../knowledge/architecture/automation-framework-survey.md — 需要回看 ok-script / MaaFramework / Airtest / RPA 调研结论。
- ../../knowledge/architecture/target-controller-upgrade-guide.md — 需要回看长期升级路线、Go/Rust 分工、Android/Win32/Browser 策略。
- ../../knowledge/nodes/node-system-architecture.md — 迁移节点或 runtime service 前。
- ../../knowledge/subgraph/asset-subsystem.md — 改截图取点、资产 capture、模板变体前。
- ../../knowledge/input/sendinput-primitive-size-and-return.md — 调 Win32 SendInput primitive 前。

## Progress

Done:
- 市面框架调研。
- 总体设计 spec。
- Phase 1 implementation plan。
- cockpit 中加入恢复入口。
- Phase 1 代码：`internal/automation/target`、`internal/automation/controller`、runtime WindowHandle -> Target bridge。
- Phase 2 代码：`internal/automation/trace`、Win32Controller 可选 controller-call trace hook。
- Phase 3 代码：`RuntimeContext` 拥有 per-run trace recorder，并提供 `TraceRecorder` / `TraceRecords` / `ClearTrace`。
- Phase 4 代码：`InputService.KeyDown/KeyUp` 经 `Win32Controller` 执行，并写入 runtime trace。
- Phase 5 代码：`InputService.Click` 经 `Win32Controller` 执行，并写入 runtime trace。
- Phase 6 代码：`InputService.MoveTo` 经 `Win32Controller` 执行，并记录最小 coordinate step。
- Phase 7 代码：`InputService.Scroll` 经 `Win32Controller` 执行，并记录最小 coordinate step。
- Phase 8 代码：controller action trace 增加 `ActionSource`，framework dispatch 的输入动作带 container/node/kind/in-pin 来源。
- Phase 9 代码：`InputService.TypeText` 经 `Win32Controller.Text` 执行，并写入带 source 的 `text` trace。
- Phase 10 代码：`InputService.MouseDown/MouseUp/Drag` 经 `Win32Controller` 执行，并写入带 source 的 mouse/drag trace。

Current:
- 规划 Phase 11：评估并迁移或隔离 `MouseMoveRel`。

## Open questions

- Phase 3 不改变节点路由；真正把节点动作通过 Win32Controller 执行要另写 Phase 4 plan。
- Phase 8 source metadata 覆盖 framework dispatch 的 input action；直接 `NewInputAdapter(rt)` 调用仍保持空 source。
- Remaining input methods not routed through `Win32Controller`: MouseMoveRel.
