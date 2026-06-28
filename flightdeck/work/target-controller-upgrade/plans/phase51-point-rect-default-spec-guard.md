# Phase 51 — Point/Rect Default Spec Guard

## Problem

The node spec knowledge still claimed Point/Rect defaults should be JSON-like maps and should avoid structs. Current runtime default handling keeps Go spec defaults as Go values, and `Inputs.Point` / `Inputs.Rect` read `node.Point` / `node.Rect` directly. The knowledge was stale and there was no guard against future mismatches.

## Scope

- Add a Go spec consistency test for Point/Rect defaults.
- Refresh node spec knowledge to use `node.Point` / `node.Rect` defaults.
- Keep nil Point/Rect defaults valid as "no default".

## Non-goals

- No container literal validation change.
- No migration of runtime config/literal map decoding.
