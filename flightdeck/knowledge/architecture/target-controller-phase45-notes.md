# Target Controller Upgrade — Phase 45 Notes

## Completed

- Removed the lowercase `out` / `in` fallback from frontend edge persistence.
- Added a regression test proving missing handles do not create edges.
- Updated node spec knowledge to replace the stale `useFlowInteraction.ts` TODO with the current `onConnect` guard.

## Verification

- `pnpm vitest run src/composables/containerEditor/useGraphMutations.test.ts`
- `pnpm vue-tsc --noEmit`
- `pnpm test`

## Result

VueFlow connection events now fail closed when handles are missing. Invalid lowercase edge refs should not be silently written into container graphs from the frontend connection path.
