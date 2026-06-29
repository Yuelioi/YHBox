# Target Controller Upgrade — Phase 49 Notes

## Completed

- `useInlineMenu.isValidVueFlowConnection` now rejects connections missing either VueFlow handle.
- Added a Vitest regression covering missing `sourceHandle` and missing `targetHandle`.
- This aligns the drag-time validation layer with the persisted `onConnect` guard added in Phase 45.

## Verification

- `cd frontend && ./node_modules/.bin/vitest run src/composables/containerEditor/useInlineMenu.test.ts`

## Result

Frontend graph connections now fail closed consistently before and during persistence when VueFlow does not provide explicit handle IDs.
