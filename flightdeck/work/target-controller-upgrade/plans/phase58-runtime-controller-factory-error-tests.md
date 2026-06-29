# Phase 58 — Runtime Controller Factory Error Tests

## Problem

Runtime active-target paths already rejected missing controller factories, but injected factory failures were not explicitly guarded. These errors must propagate cleanly and must not fall back to stale Win32 window input/capture backends.

## Scope

- Add input adapter coverage for controller factory errors.
- Add vision adapter coverage for controller factory errors.
- Assert the active target is still passed into the factory.

## Non-goals

- No runtime controller factory behavior change.
- No frontend error presentation change.
