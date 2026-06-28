# Phase 50 — Bool Default Spec Guard

## Problem

Node spec consistency guarded numeric defaults and string defaults, but Bool inputs could still accidentally use a string/number default. That would be hard to notice in review because defaults are declared across many node packages.

## Scope

- Add a Go spec consistency test: Bool defaults, when set, must be Go `bool`.
- Keep nil Bool defaults valid as "no default".
- Refresh flightdeck state for the new guard.

## Non-goals

- No node spec migration unless the new guard finds an existing violation.
- No runtime input coercion change.
