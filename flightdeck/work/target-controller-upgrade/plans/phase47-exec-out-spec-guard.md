# Phase 47 — Exec-Out Spec Naming Guard

## Problem

The node spec style guide required PascalCase exec output pins, but the Go guard only checked data pins and exec input pins. A new node could still introduce lowercase `out` / `done` / `yes` / `no` without failing tests.

## Scope

- Add a Go spec consistency test for exec output pin names.
- Allow only the intentional Switch `default` output as a lowercase reserved exit.
- Update node spec knowledge so the documented guard matches the code.

## Non-goals

- No node spec changes; current registered nodes already satisfy the rule.
