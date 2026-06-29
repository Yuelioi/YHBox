# Phase 35 — Fast Runtime State Tests

## Problem

`go test ./internal/services/container/runtime` spends about two minutes in fishing-v2 state-machine tests. The slow path is not production logic complexity; tests load production JSON fixtures whose `Sleep`, `ClickTemplate.TimeoutMs`, and `KeyPress.DurationMs` values are intentionally human-scale.

## Decision

Keep production fixtures unchanged. In tests only, cap timing literals after JSON load so the same graph topology and routing are exercised without waiting for production delays.

## Scope

- Add a runtime test helper that caps fixture timing literals in memory.
- Apply it to fishing-v2 state/helper fixtures used by slow tests.
- Verify targeted slow tests and the full runtime package.

## Non-goals

- Do not edit `testdata/fishing-v2` JSON.
- Do not weaken state assertions or remove loop coverage.
