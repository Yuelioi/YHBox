# Phase 52 — Frontend Build Warning Triage

## Problem

`pnpm build` was green but emitted an ineffective dynamic import warning because `pinSpec.ts` was already statically imported by several modules while `nodeRegistry/adapter.ts` still dynamically imported it. The warning made the build baseline noisy and hid more meaningful future warnings.

## Scope

- Replace the ineffective dynamic imports for rebuild hooks with static imports.
- Verify production build still succeeds.
- Document the remaining non-blocking build warnings.

## Non-goals

- No bundle splitting refactor.
- No Nuxt UI / Wails plugin performance tuning.
