# Phase 18 — Controller Backend Profiles

## Goal

Add a code-level backend capability/profile matrix so Win32, Browser CDP, Android ADB, mock, and replay backends have one reusable source of truth before concrete Browser/Android controllers are implemented.

## Scope

- Define backend identifiers and profiles in `internal/automation/controller`.
- Map each backend to supported target kinds, coordinate spaces, and default capabilities.
- Add lookup helpers by backend and by target kind.
- Add tests for Win32, Browser CDP, Android ADB, unknown backend, and unknown target kind.

## Non-goals

- Implement Browser CDP or Android ADB input transport.
- Add frontend UI for selecting the backend matrix.
- Change runtime routing.

## Verification

- `go test ./internal/automation/controller -count=1`
