# Target / Controller Phase 17 Notes

SUMMARY: Phase 17 persists redacted action traces into the existing daily JSONL log file
READ WHEN: Debugging post-mortem action trace logs or changing log persistence
RECHECK WHEN: `App.Emit`, `LogSink`, `container:action-trace`, or trace payload fields change

---

Phase 17 adds action trace persistence through the existing log file path:

- `App.Emit("container:action-trace", data)` now calls `LogSink.AppendActionTrace(data)` before forwarding the live frontend event.
- `shouldMirrorToRootLog` excludes `container:action-trace` so generic root logging cannot persist raw `request` / `result` payloads.
- `LogSink.AppendActionTrace` writes a file-only JSONL row when file logging is enabled.
- The persisted row is a whitelist: event, containerId, action, source, target id/kind, backend, status, error, coordinate step count, startedAt, endedAt, durationMs, time.
- Raw `request`, `result`, coordinate payloads, and target OS handles are not persisted.

Verification:

- `go test ./internal/services -run "TestLogSink_AppendActionTrace|TestLogSink_AppendDumpLine|TestShouldMirrorToRootLog" -count=1`
- `go test ./internal/services -count=1`

Still not covered:

- UI for browsing historical log files.
- Per-container trace retention limits beyond the daily log file.
- Opt-in raw payload capture for local-only deep debugging.
