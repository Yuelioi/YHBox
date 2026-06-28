# Phase 3 Runtime Trace Ownership Plan

## Goal

Give each `RuntimeContext` its own trace recorder and expose safe accessors for later node/controller routing.

## Scope

In scope:

- `RuntimeContext` owns a `trace.MemoryRecorder`.
- Runtime exposes `TraceRecorder()`, `TraceRecords()`, and `ClearTrace()`.
- Runtime tests prove records are per-runtime, copied on read, and clearable.
- Documentation states this phase does not route node actions through controllers yet.

Out of scope:

- Changing existing nodes to use controllers.
- Feature flags.
- Trace UI or Wails service.
- Disk persistence.
- Android/browser work.

## Task 1: Runtime Trace Recorder

Files:

- Modify `internal/services/container/runtime/runtime_context.go`
- Create `internal/services/container/runtime/runtime_trace_test.go`

Steps:

1. Write tests:
   - `TestRuntimeContextTraceRecorderStoresRecords`
   - `TestRuntimeContextTraceRecordsReturnsCopy`
   - `TestRuntimeContextTraceIsPerRuntime`
   - `TestRuntimeContextClearTrace`
2. Add `traceRecorder *trace.MemoryRecorder` to `RuntimeContext`.
3. Initialize it in `NewRuntimeContext`.
4. Add methods:
   - `TraceRecorder() trace.Recorder`
   - `TraceRecords() []trace.ActionRecord`
   - `ClearTrace()`
5. Run `go test ./internal/services/container/runtime -run TestRuntimeContextTrace -count=1`.
6. Commit as `feat(runtime): own automation trace recorder`.

Acceptance:

- `TraceRecords()` returns a copy.
- Separate runtime contexts do not share records.
- `ClearTrace()` removes records for that context only.

## Task 2: Notes And Verification

Files:

- Create `flightdeck/knowledge/architecture/target-controller-phase3-notes.md`
- Modify `flightdeck/work/target-controller-upgrade/index.md`

Steps:

1. Document that runtime now owns trace, but nodes still do not route through controllers.
2. Update topic index to mark Phase 3 complete.
3. Run:

```powershell
go test ./internal/automation/... -count=1
go test ./internal/services/container/runtime -run "TestRuntimeContextTrace|TestWindowHandleToTarget" -count=1
```

4. Commit as `docs(architecture): record runtime trace ownership phase`.

Acceptance:

- Worktree is clean after commits.
- Next topic step says to write a Phase 4 plan before routing a node through Win32Controller.

