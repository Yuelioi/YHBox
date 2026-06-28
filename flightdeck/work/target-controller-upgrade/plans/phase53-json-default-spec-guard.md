# Phase 53 — JSON Default Spec Guard

## Problem

JSON input defaults were documented as object defaults, but the spec consistency tests did not enforce the type. A future node could accidentally use a string or array default and only fail later in editor/runtime paths.

## Scope

- Add a Go spec consistency test for JSON defaults.
- Keep nil JSON defaults allowed only when the node intentionally has no default.
- Refresh node spec knowledge and flightdeck state.

## Non-goals

- No JSON schema validation expansion.
- No migration of existing JSON inputs unless the guard finds a violation.
