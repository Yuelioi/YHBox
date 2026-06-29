# Target / Controller Phase 2 Notes

SUMMARY: Phase 2 adds controller-call trace records and an in-memory recorder; runtime node trace is not integrated yet
READ WHEN: Continuing trace/report work / wiring controller calls into runtime / debugging why UI cannot show traces yet
RECHECK WHEN: `internal/automation/trace` or controller trace hooks change

---

Phase 2 adds the first trace foundation:

- `internal/automation/trace` defines `ActionRecord`, `Status`, `CoordinateStep`, `Recorder`, and `MemoryRecorder`.
- `controller.Win32Deps` has optional `Trace` and `Backend` fields.
- `Win32Controller` records Click, Move, Scroll, KeyChord, KeyDown, KeyUp, Text, and Screenshot calls when a recorder is provided.

This phase does **not** integrate trace with node runtime yet:

- No node id / container id is recorded.
- No UI trace viewer exists.
- No file persistence exists.
- No screenshot before/after capture is attached to trace.
- Existing nodes still call current runtime services directly unless later routed through controllers.

Current guarantee:

- Controller behavior is unchanged when `Win32Deps.Trace` is nil.
- Successful calls record `StatusSuccess`.
- Failed calls record `StatusError` and the same error string returned to the caller.
- Empty backend labels default to `win32`.

Next phase candidates:

- Add a runtime-owned trace recorder to `RuntimeContext`.
- Route one narrow action path through `Win32Controller` behind a feature flag.
- Attach node id / container id around controller calls.
- Add a minimal trace dump endpoint before building UI.
