# Phase 33 — Widget props validation

## Goal

Make node widget declarations fail fast when their loosely typed `WidgetSpec.Props` drift from the frontend/runtime contract.

## Scope

- Add spec consistency tests for supported widget kinds.
- Validate required props for structured widgets:
  - `dropdown` requires non-empty `options`.
  - `slider` requires sane numeric `min/max/step`.
  - `async-dropdown` requires `asyncSource`.
  - `async-dropdown.applyMeta` must map non-empty meta keys to existing non-exec input pins on the same node.
- Keep this as a test guard; do not change runtime behavior unless the guard exposes current drift.

## Verification

- `go test ./internal/node -count=1`
- Broader affected node tests if the guard forces spec changes.
