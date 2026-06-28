# Index — target-controller-upgrade

## State

破坏性大升级 topic。调研、总体设计、Phase 1 抽象层、Phase 2 controller-call trace foundation、Phase 3 runtime trace ownership 已完成并提交。核心决策：Go 保持主运行时，Rust 只作为 Win32/native controller hot path；先引入 `Target / Controller / CoordinateSpace / Trace`，再迁移节点、Android、浏览器和输入后端矩阵。

## Next

Execute `plans/phase4-keyboard-controller.md`: route only `InputService.KeyDown/KeyUp` through `Win32Controller` and runtime trace. Do not migrate mouse/click/text/screenshot until a later plan.

## Read now

- design.md
- plan.md
- plans/phase2-trace.md
- plans/phase3-runtime-trace.md
- plans/phase4-keyboard-controller.md
- ../../knowledge/architecture/target-controller-phase3-notes.md

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

Current:
- 执行 Phase 4 keyboard controller routing：先迁移 `InputService.KeyDown/KeyUp`。

## Open questions

- Phase 3 不改变节点路由；真正把节点动作通过 Win32Controller 执行要另写 Phase 4 plan。
