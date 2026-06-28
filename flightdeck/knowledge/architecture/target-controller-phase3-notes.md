# Target / Controller Phase 3 Notes

SUMMARY: Phase 3 makes RuntimeContext own a per-run trace recorder; nodes still do not route through controllers
READ WHEN: Continuing runtime trace work / exposing trace to services or UI / planning node migration to Win32Controller
RECHECK WHEN: `RuntimeContext` trace accessors or controller routing change

---

Phase 3 gives each runtime instance its own in-memory trace owner:

- `RuntimeContext` stores a `trace.MemoryRecorder`.
- `TraceRecorder()` returns the runtime recorder as `trace.Recorder`.
- `TraceRecords()` returns a copy of records for safe inspection.
- `ClearTrace()` clears only that runtime context.

Current guarantee:

- Separate `RuntimeContext` instances do not share trace records.
- Reading records cannot mutate stored trace state.
- Zero-value/runtime-test contexts lazily create a recorder when the accessors are called.

This phase does **not** route node actions through controllers yet:

- Existing click/input/capture nodes still call their current runtime services.
- No node id, pin id, or container id is attached to trace records yet.
- No trace UI, Wails service, or disk persistence exists.
- Android, browser/CDP, and feature-flag work remain out of scope.

Next phase should start with a new plan before code changes:

- Choose one narrow Win32 node/action path to route through `Win32Controller`.
- Attach runtime-owned trace to that controller call.
- Define the minimum node/context metadata that must enter `ActionRecord`.
