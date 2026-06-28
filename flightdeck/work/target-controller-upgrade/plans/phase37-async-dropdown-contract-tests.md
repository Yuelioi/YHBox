# Phase 37 — Async Dropdown Contract Tests

## Problem

Backend now validates async source registration and frontend adapter preserves `asyncSource/applyMeta`, but the behavioral contract still has gaps:

- `PinInput` must call `NodeService.AsyncOptions` with node/context inputs.
- Selecting an async option must emit both the coerced pin value and option metadata.
- Inspector metadata application must write selected value plus non-empty mapped meta fields into sibling literals.

## Scope

- Add focused frontend tests.
- Extract Inspector metadata merge into a small pure helper.

## Non-goals

- No UI redesign.
- No backend behavior changes.
