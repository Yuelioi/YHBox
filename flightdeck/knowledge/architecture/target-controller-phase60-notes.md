# Target Controller Upgrade — Phase 60 Notes

> Superseded by Phase 61: target palette split still stands, but the later naming migration became destructive. Do not keep old `WindowTarget` readable/runnable.

## Completed

- Moved target selection nodes from backend category `Window` to `Target`:
  - `Win32WindowTarget`
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

This section is obsolete after Phase 61. Continue with contract hardening around `Win32WindowTarget` and `NeedsTarget`; do not add old-kind compatibility.
