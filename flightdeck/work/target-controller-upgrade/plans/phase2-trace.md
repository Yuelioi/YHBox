# Phase 2 Trace Implementation Plan

## Goal

Add the first trace foundation for controller calls: data records, an in-memory recorder, and optional Win32Controller action recording.

## Scope

In scope:

- `internal/automation/trace` package.
- `trace.ActionRecord` and `trace.MemoryRecorder`.
- Optional `Trace` field on `controller.Win32Deps`.
- Win32Controller records Click, Move, Scroll, KeyChord, KeyDown, KeyUp, Text, and Screenshot calls.
- Tests prove successful and failed controller calls are recorded.

Out of scope:

- Trace UI.
- Persisting trace to disk.
- Runtime node trace integration.
- Action router.
- Android/browser controllers.
- Screenshot before/after capture.

## Task 1: Trace Records And Memory Recorder

Files:

- Create `internal/automation/trace/trace.go`
- Create `internal/automation/trace/memory.go`
- Create `internal/automation/trace/memory_test.go`

Steps:

1. Write tests for `MemoryRecorder.Record`, `Records`, and `Clear`.
2. Implement `ActionRecord`, `Status`, `CoordinateStep`, and `Recorder`.
3. Implement a mutex-protected `MemoryRecorder`.
4. Run `go test ./internal/automation/trace -count=1`.
5. Commit as `feat(automation): add trace recorder`.

Acceptance:

- Records preserve action, target id, backend, status, error message, and timing.
- `Records()` returns a copy, not the mutable backing slice.

## Task 2: Win32Controller Trace Hook

Files:

- Modify `internal/automation/controller/win32.go`
- Modify `internal/automation/controller/win32_test.go`

Steps:

1. Add tests proving `Click` records a successful trace.
2. Add tests proving `KeyChord` records an error trace when `KeyDown` fails.
3. Add optional `Trace trace.Recorder` and `Backend string` to `Win32Deps`.
4. Add a private helper that records target, backend, action, request, status, timing, and error.
5. Route all Win32Controller public action methods through the helper.
6. Run `go test ./internal/automation/controller -count=1`.
7. Commit as `feat(automation): trace win32 controller actions`.

Acceptance:

- Existing controller behavior stays unchanged when no recorder is provided.
- Default backend label is `win32` when `Win32Deps.Backend` is empty.
- Error traces include the returned error string and still return the same error to caller.

## Task 3: Notes And Verification

Files:

- Create `flightdeck/knowledge/architecture/target-controller-phase2-notes.md`
- Modify `flightdeck/work/target-controller-upgrade/index.md`

Steps:

1. Document that Phase 2 is controller-call trace only, not runtime node trace.
2. Update topic index progress and next step.
3. Run:

```powershell
go test ./internal/automation/... -count=1
go test ./internal/services/container/runtime -run TestWindowHandleToTarget -count=1
```

4. Commit as `docs(architecture): record target controller trace phase`.

Acceptance:

- Worktree is clean after commits.
- Topic index says Phase 1 complete and Phase 2 complete if all tasks pass.

