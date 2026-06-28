# Phase 16 — Action Trace Drawer

## Goal

Expose the structured `container:action-trace` stream in the frontend as a compact timeline/debug drawer, so click/input/capture failures can be inspected without reading only the compressed log line.

## Scope

- Add a lightweight action trace viewer opened from `LogPanel`.
- Back the viewer directly by `useLogStore().actionTraces`.
- Keep the UI dense: action, status, source node, target, backend, duration, coordinate step count, error.
- Add localized labels for the new controls.
- Keep existing log line behavior unchanged.

## Non-goals

- Persist traces to disk.
- Add backend pagination or batching.
- Add a full run-history debugger.

## Verification

- `pnpm vue-tsc --noEmit`
- Existing focused log store tests if touched.
