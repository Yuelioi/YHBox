# Phase 55 — Controller Coordinate Boundary Tests

## Problem

Android ADB and Browser CDP controllers had normal-path coordinate tests, but boundary behavior was not explicitly guarded: normalized edge clamping, native coordinate passthrough, unsupported coordinate spaces, and missing resolution failures.

## Scope

- Add Android ADB controller tests for coordinate boundary conversion.
- Add Browser CDP controller tests for coordinate boundary conversion.
- Assert unsupported coordinate spaces fail before issuing backend calls.

## Non-goals

- No coordinate conversion behavior change.
- No frontend coordinate UI change.
