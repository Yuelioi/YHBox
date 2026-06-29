# Target Controller Upgrade — Phase 59 Notes

> Superseded by Phase 61: 项目未上线，已执行破坏性命名更新。旧 `WindowTarget` 不再支持，后续不得按本页早期“alias/兼容”建议执行。

## Completed

- Added the phase59 implementation plan for Window/Target terminology cleanup.
- Added `window-vs-target-boundary.md` as the architecture boundary note.
- Updated frontend i18n so user-facing copy distinguishes:
  - generic automation target / target frame
  - Windows HWND window
  - legacy-compatible `Win32WindowTarget` displayed as Windows window target
- Updated Go comments so future node authors do not treat `NeedsWindow` or `BringWindowForeground` as generic Android/Browser target concepts.

## Boundary

`Win32WindowTarget` remains the serialized compatibility kind. This phase deliberately did not rename old container JSON, validator internals, MCP examples, or Go function names.

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

This section is obsolete after Phase 61. Do not add a compatibility alias for old `WindowTarget`; keep moving toward `NeedsTarget(kind=win32-window, capabilities=...)`.
