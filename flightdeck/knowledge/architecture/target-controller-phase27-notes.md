# Target Controller Upgrade — Phase 27 Notes

## Completed

- Frontend `FieldSchema` now carries `asyncSource`.
- Backend `WidgetSpec{Kind:"async-dropdown", Props:{asyncSource}}` is preserved by the node registry adapter.
- `NodeInspector` passes node ID, kind, and current config/literal inputs into `PinInput`.
- `PinInput` renders `async-dropdown` with `UInputMenu`:
  - calls `NodeService.AsyncOptions`
  - shows loading/error state
  - keeps manual text entry fallback
  - guards against stale async responses when switching nodes
- Added adapter coverage for `AndroidTarget.Serial -> androidADBDevices`.

## Boundary

`node.EnumOption` currently has only value/label. That is enough for Android serial selection, but not enough for richer device UX such as disabled offline rows, status badges, or carrying width/height metadata back into sibling fields.

Width/height auto-fill remains intentionally out of scope for this phase. It should be handled by adding richer option metadata or a dedicated inspector affordance, not by parsing label text.

## Verification

- `pnpm vitest run src/components/containers/nodeRegistry/adapter.test.ts`
- `pnpm vue-tsc --noEmit`

`vitest` emitted a `127.0.0.1:3000` connection-refused aggregate after the passing summary, but exited 0. Treat as existing test-environment noise unless it starts failing the run.

## Next Risk

Browser/CDP target discovery and client lifecycle are still skeleton-only. The next slice should make browser targets discoverable and ensure runtime controller construction gets a live CDP client instead of returning the current "client wiring not installed" path.
