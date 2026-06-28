# Phase 57 — Browser Controller Nil Client Tests

## Problem

Runtime factory wiring normally prevents Browser CDP controllers without clients, but direct controller construction still has a clear nil-client failure mode. Health checks and action traces should report that state consistently.

## Scope

- Add Browser CDP controller health-check coverage for nil client.
- Add Browser CDP controller action trace coverage for nil-client errors.

## Non-goals

- No runtime factory behavior change.
- No Browser CDP provider lifecycle change.
