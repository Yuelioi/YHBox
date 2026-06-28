# Target Controller Upgrade — Phase 59 Notes

## Completed

- Added the phase59 implementation plan for Window/Target terminology cleanup.
- Added `window-vs-target-boundary.md` as the architecture boundary note.
- Updated frontend i18n so user-facing copy distinguishes:
  - generic automation target / target frame
  - Windows HWND window
  - legacy-compatible `WindowTarget` displayed as Windows window target
- Updated Go comments so future node authors do not treat `NeedsWindow` or `BringWindowForeground` as generic Android/Browser target concepts.

## Boundary

`WindowTarget` remains the serialized compatibility kind. This phase deliberately did not rename old container JSON, validator internals, MCP examples, or Go function names.

Win32-only concepts:

- `WindowService`
- `WindowHandle`
- `WindowInputSpec`
- `NeedsWindow`
- `BringWindowForeground`

Generic target concepts:

- `TargetService`
- `target.Target`
- controller factory and capability checks
- Android/Browser target nodes

## Verification

- `go test ./internal/node ./internal/nodes/input`
- `cd frontend && pnpm i18n:check`

## Next Risk

The next migration should not start with a hard rename. Add a compatibility alias first:

- new display-facing `Win32WindowTarget` or `WindowsWindowTarget`
- loader/validator maps old `WindowTarget` to the same internal Win32 target meaning
- `NeedsWindow` begins migrating toward `NeedsTarget(kind=win32-window, capabilities=...)`
- frontend palette can later prefer the alias while still rendering old containers
