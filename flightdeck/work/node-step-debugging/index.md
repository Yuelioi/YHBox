# Index — node-step-debugging

## State

Ready for human review. V1 node step debugging is wired through runtime, service/RPC, Wails bindings, editor controls, right-click Debug from here, debug state events, node highlights, editor state resync, Wails event payload normalization, quiet run/debug success feedback, and terminal debug panel cleanup.

## Next

Human review smoke-test:
- Toolbar Debug starts from the graph entry.
- Right-click a node and choose Debug from here.
- Use Step / Continue / Pause / Stop and verify node highlights match execution.

## Read now

- `design.md` — V1 node step debugging scope, runtime semantics, UX, validation, and test plan.
- `implementation.md` — implementation file map, task breakdown, and verification commands.

## Read if

- `flightdeck/work/window-control/design.md` — useful reference for how prior runtime-affecting node work was structured.
- `flightdeck/knowledge/build/build.md` — build and smoke-test expectations before shipping implementation.

## Progress

Done:
- Defined V1 around real token-step debugging: Debug from entry, Debug from here, Step, Continue, Pause, Stop.
- Clarified that V1 Step means one outer runtime `ExecToken`; `Subgraph`, `CollapsedNode`, `Loop`, `Parallel`, and `Race` are atomic region nodes in V1.
- Defined debug session DTOs, token summaries, warnings, in-memory storage, execution lease, command concurrency, and live update events.
- Defined startup validation vs runtime context failures, including `GraphPath`, invalid selected nodes, and listener-driven nodes.
- Added concrete backend, frontend, and integration test sequences.
- Deferred breakpoints, selected-fragment runs, explicit subgraph step into/over, and mock external inputs to later phases.
- Explicitly rejected `EventTick` listener debugging in V1.
- Implemented runtime lifecycle and one-token stepping primitives.
- Added process-local debug sessions with execution exclusivity against normal runs.
- Added service/RPC DTOs and regenerated Wails bindings.
- Added Pinia debug state, toolbar controls, right-click node action, and canvas highlights for next/running/last/failed nodes.
- Added an in-canvas debug status panel for current/next node, last exit, queue count, warnings, and errors.
- Added Debug from here confirmation copy and hid the action for nodes without exec inputs.
- Added visible side-effect warnings near debug controls and in the debug status panel.
- Added frontend stale-session recovery: failed debug RPC responses clear local debug UI state, and failed starts no longer show success.
- Added debug manager tests for second-start and duplicate-step busy rejection.
- Added compact debug panel previews for last output data and variable snapshots.
- Added compact debug panel preview for queued debug tokens.
- Added debug manager test for stopping a long-running step and restarting without a leaked session.
- Added editor mount/activation debug state resync through `DebugState`.
- Added debug panel close control to stop active sessions or clear retained finished/failed results.
- Fixed Wails `debug:state` event field normalization so async Step completion restores the toolbar to paused/next-step state.
- Added regression coverage for stepping from `AndroidTarget` to the following Android app node.
- Fixed Wails event payload unwrapping for nested `data`/array event shapes so real debug completion events are applied after a step.
- Removed success toasts for normal run enqueue and debug session start; visible UI state is the feedback.
- Fixed out-of-order debug command snapshots so a late `stepping` RPC response cannot overwrite a newer paused completion event.
- Moved validate/save into the centered run/debug workflow group and removed the standalone warning icon beside Debug.
- Skipped disabled nodes at debug queue boundaries so Step pauses on the next enabled node instead of a disabled passthrough node.
- Reworked the editor toolbar into IDE-style zones: navigation on the left, save/validate/run/debug workflow in the center, recording/layout/more tools on the right.

Verified:
- Design self-review completed for terminology, scope, API, validation, edge cases, and test coverage.
- `go test ./internal/services/container/runtime -run TestDebug -count=1`
- `go test ./internal/services/container -run TestServiceDebug -count=1`
- `go test . -run TestDebugManager -count=1`
- `go test . -run TestDebugManagerStepAndroidTargetPausesAtNextNode -count=1`
- `go test ./internal/services/container/runtime -run "TestDebug(SeedFromEntrySkipsDisabledHead|StepOnceSkipsDisabledDownstreamNode)" -count=1`
- `go test . -run "TestDebugManagerStepSkipsDisabledDownstreamNode" -count=1`
- `go test ./...`
- `pnpm exec vitest run src/stores/execution.debug.test.ts`
- `pnpm exec vitest run src/components/containers/ContainerEditorToolbar.spec.ts`
- `pnpm exec vitest run src/composables/editor/debugPanel.spec.ts`
- `pnpm exec vitest run`
- `pnpm exec vitest run src/components/containers/menus/NodeContextMenu.spec.ts`
- `pnpm exec vue-tsc --noEmit`
- `pnpm exec node src/i18n/check.cjs`
- `wails3 generate bindings ./...`
- `pnpm exec vite build --mode production`
- `wails3 build`
