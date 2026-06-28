# Target Controller Upgrade — Phase 60 Notes

## Completed

- Moved target selection nodes from backend category `Window` to `Target`:
  - `WindowTarget`
  - `AndroidTarget`
  - `BrowserTarget`
- Added frontend `target` group support:
  - backend `Target` category mapping
  - `NodeGroup` union
  - group ordering
  - group label i18n
  - group visual icon/color
- Added a Go guard so target selection nodes do not drift back into the Window category.

## Boundary

This phase only changes palette/category metadata. It does not rename serialized node kinds and does not migrate container JSON.

`Window` category is now reserved for Windows HWND operations such as:

- wait/get/close/move/resize/window-state
- bring foreground

`Target` category is for selecting the current automation object:

- Windows HWND target
- Android ADB target
- Browser CDP target

## Verification

- `go test ./internal/nodes/system ./internal/catalog`
- `cd frontend && pnpm i18n:check`
- `cd frontend && pnpm vue-tsc --noEmit`

## Next Risk

The next naming migration can add a `Win32WindowTarget` alias, but it should keep `WindowTarget` readable and runnable until a loader migration and old-container test exist.
