# Phase 45 — Frontend Edge Handle Guard

## Problem

Frontend graph connection code still guessed missing VueFlow handles as lowercase `out` / `in`. The backend pin convention is PascalCase (`In`, `Done`, semantic exits), so a missing handle could persist an invalid edge instead of failing closed.

The older knowledge note mentioning `useFlowInteraction.ts` is stale because that file no longer exists, but the underlying "do not guess lowercase pins" rule is still valid.

## Scope

- Change `useGraphMutations.onConnect` to reject connections with missing source/target handles.
- Add a Vitest regression for the missing-handle case.
- Refresh node spec knowledge to remove the stale `useFlowInteraction.ts` note and document the current guard.

## Non-goals

- No drag-to-edge insertion feature work.
- No change to valid VueFlow connections with real handles.
