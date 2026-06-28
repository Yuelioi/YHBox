# Phase 48 — String Default Spec Guard

## Problem

The node spec style guide claimed String defaults must be non-nil, but current specs intentionally leave required string inputs without defaults so validation can enforce required fields. The useful guard is not "non-nil"; it is "if a String default is set, it must actually be a string".

## Scope

- Add a Go spec consistency test for String default types.
- Update node spec knowledge to reflect nil-as-no-default semantics.

## Non-goals

- No node spec migration.
- No change to Required validation behavior.
