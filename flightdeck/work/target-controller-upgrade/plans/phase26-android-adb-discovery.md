# Phase 26 — Android ADB Discovery

## Goal

Provide a reusable backend discovery service for connected Android ADB devices so `AndroidTarget` no longer depends only on hand-entered serial/resolution.

## Scope

- Add a small Android ADB discovery package/service.
- Parse `adb devices` output.
- Query device resolution via `adb -s <serial> shell wm size`.
- Return stable target-shaped metadata.
- Register a `NodeService` async source for Android devices.
- Point `AndroidTarget.Serial` at that async source in widget metadata.

## Non-goals

- No frontend generic async-dropdown implementation in this phase.
- No emulator launch/management.
- No app package discovery.
- No CDP discovery.

## Verification

- Unit tests for ADB device and resolution parsing.
- Unit test for async option registration/shape.
- `go test ./internal/services/androidadb ./internal/node ./internal/nodes/system -count=1`

