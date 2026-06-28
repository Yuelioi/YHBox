# Phase 49 — Inline Handle Validation

## Problem

The persisted graph mutation path already rejects VueFlow connections that do not include both handle IDs, but the inline drag validation still treated missing handles as valid. That made the UI preview layer less strict than the save path and left room for inconsistent behavior when VueFlow emits partial connection data.

## Scope

- Make inline connection validation fail closed when either handle is missing.
- Add a Vitest regression for missing `sourceHandle` / `targetHandle`.
- Refresh flightdeck state so this slice is resumable.

## Non-goals

- No change to valid typed connection compatibility checks.
- No node spec migration.
