# Phase 34 — Granular controller capabilities

## Goal

Make target/controller compatibility explicit for pointer actions that currently share broad interfaces but are not supported by every backend.

## Scope

- Extend `controller.CapabilitySet` with granular pointer capabilities:
  - mouse button hold/release
  - drag
  - relative move
- Update backend profiles and controller capability reports.
- Add runtime adapter checks before invoking each action.
- Add tests for capability/profile consistency and unsupported target/action failures.

## Verification

- `go test ./internal/automation/controller ./internal/services/container/runtime -count=1`
