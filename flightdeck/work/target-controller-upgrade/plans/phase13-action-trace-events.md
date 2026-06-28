# Phase 13 Action Trace Event Surface Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Emit controller action trace records through the runtime event surface so UI/debug tooling can consume them without reaching into `RuntimeContext`.

**Architecture:** Wrap the per-runtime memory trace recorder with an emitting recorder returned by `RuntimeContext.TraceRecorder()`. The wrapper records to memory first, then emits a normalized `container:action-trace` payload through `RuntimeContext.Emit` when available. Existing source-enriching wrappers remain outside this recorder, so source metadata is filled before the event is emitted.

**Tech Stack:** Go, runtime trace recorder, event payload tests.

---

## Scope

In scope:

- Add an emitting trace recorder wrapper in runtime.
- Emit `container:action-trace` for every controller action trace record.
- Keep `TraceRecords()` and `ClearTrace()` behavior unchanged.
- Add tests for event payload and memory recorder behavior.

Out of scope:

- Frontend trace viewer.
- Trace persistence.
- Event throttling/batching.
- Redaction policy for large request/result payloads.

## Event Payload

Event name:

```text
container:action-trace
```

Payload fields:

- `containerId`
- `action`
- `source`
- `target`
- `backend`
- `request`
- `result`
- `status`
- `error`
- `coordinateSteps`
- `startedAt`
- `endedAt`
- `durationMs`

## Tasks

1. Commit this plan and update `flightdeck/work/target-controller-upgrade/index.md`.
2. Add failing runtime trace test for `container:action-trace` emission.
3. Implement emitting recorder wrapper and payload builder.
4. Run focused trace/runtime tests.
5. Add Phase 13 notes, update index, and commit docs.

## Verification

```powershell
go test ./internal/services/container/runtime -run "TestRuntimeContextTrace|TestExecNodeViaFramework_Capture" -count=1
```

## Acceptance

- `TraceRecorder().Record` still stores records in memory.
- If `RuntimeContext.Emit` is non-nil, every trace record emits `container:action-trace`.
- Source metadata added by per-node service wrappers appears in the emitted payload.
- Worktree is clean after commits.
