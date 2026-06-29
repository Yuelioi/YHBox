# Target / Controller Phase 13 Notes

SUMMARY: Phase 13 emits controller action traces as runtime events
READ WHEN: Building trace UI/debug tooling / changing trace payload schema / troubleshooting action trace visibility
RECHECK WHEN: `RuntimeContext.TraceRecorder`, `trace_events.go`, or event bridge code changes

---

Phase 13 adds an event surface for action traces:

- `RuntimeContext.TraceRecorder()` now returns an emitting wrapper around the per-runtime memory recorder.
- The wrapper stores the record first, then emits `container:action-trace` when `RuntimeContext.Emit` is available.
- Event payload uses stable field names instead of exposing only an opaque struct.

Event name:

- `container:action-trace`

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

Operational effect:

- UI/debug bridges can subscribe to controller action traces without reaching into `RuntimeContext.TraceRecords()`.
- Existing in-memory trace APIs still work.
- Per-node source metadata appears in the emitted event because source wrappers sit outside the runtime recorder.

Still not covered:

- Frontend trace viewer.
- Trace persistence.
- Batching/throttling and payload redaction for large request/result data.
