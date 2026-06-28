# Phase 17 — Action Trace File Persistence

## Goal

Persist action traces for post-mortem debugging without storing raw request/result payloads by default.

## Design

- Reuse the existing `LogSink` daily JSONL file writer and logger settings.
- Handle `container:action-trace` in `App.Emit` similarly to node dump:
  - keep forwarding the live event to the frontend;
  - write a sanitized JSON line to the log file when file logging is enabled.
- Store only a whitelist:
  - event, containerId, action, source, target id/kind, backend, status, error, coordinate step count, startedAt, endedAt, durationMs, time.
- Do not persist `request`, `result`, or target OS handles.

## Verification

- Add `LogSink` tests for action trace persistence and redaction.
- Add app event mirror coverage so action trace does not also mirror as a generic root log event.
- Run focused `internal/services` tests.
