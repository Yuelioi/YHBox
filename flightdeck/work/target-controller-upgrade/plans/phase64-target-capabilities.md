# Phase64 - target capability contract

## Why

Phase63 split `NeedsTarget` from direct Win32 `NeedsWindow`, but it still treats all target-aware nodes as if every target backend can perform every action. That lets invalid graphs through, for example `AndroidTarget -> MouseMoveRel`, even though the Android ADB profile has no `move-relative` capability.

## Contract

- Node specs declare the concrete target capabilities they require.
- Controller profiles remain the source of truth for what Win32 / Android ADB / Browser CDP can provide.
- Container validation checks explicit upstream target selections against node capability requirements.
- `NeedsTarget` remains the coarse "this node uses the active automation target" marker.
- Missing target defaults still use `MISSING_WIN32_WINDOW_TARGET` for the Windows-first authoring flow.

## Tasks

- [x] Add target capability metadata to node specs and catalog output.
- [x] Mark input, capture, and vision nodes with their concrete requirements.
- [x] Add graph-local target capability validation.
- [x] Add red/green tests for Android unsupported relative mouse movement.
- [x] Update Flightdeck knowledge notes.
- [x] Run Go and i18n verification.
