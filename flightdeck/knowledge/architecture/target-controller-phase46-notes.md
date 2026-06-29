# Target Controller Upgrade — Phase 46 Notes

## Completed

- Changed `pinsFor(unknownKind)` from fake lowercase `in` / `out` pins to empty pin lists.
- Added tests proving unknown kinds expose no pins while virtual subgraph markers still expose `Done` / `In`.
- Updated node spec knowledge with the stricter frontend rendering rule.

## Verification

- `pnpm vitest run src/components/containers/pinSpec.spec.ts`
- `pnpm vue-tsc --noEmit`
- `pnpm test`

## Result

Unknown frontend node kinds no longer create connectable fake handles. This removes another path for lowercase invalid edge refs to enter container graphs.
