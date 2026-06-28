# Phase 36 — Frontend Test Network Isolation

## Problem

`pnpm test` passes but emits noisy asynchronous network failures after the summary:

- `HEAD /wails/custom.js 404`
- external HTTPS timeouts from UI/icon runtime code
- happy-dom abort traces during teardown

This makes the test gate unreliable: real failures can hide inside expected noise.

## Decision

Make Vitest hermetic:

- Skip the Wails Vite plugin in test mode.
- Install a Vitest setup file that stubs network APIs with deterministic empty responses.

## Scope

- Test config only.
- No production runtime behavior change.

## Verification

- `pnpm test`
- `pnpm vue-tsc --noEmit`
