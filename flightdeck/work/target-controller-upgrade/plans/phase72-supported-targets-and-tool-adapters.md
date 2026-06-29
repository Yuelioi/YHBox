# Phase 72 — Supported Targets and Target Tool Adapter Slice

## Goal

Expose node target/platform support as derived metadata, and move editor picker
and pixel sampling entry points behind a target-aware tooling adapter boundary.

## Implemented

- Node specs exported by `NodeService.GetAllNodeSpecs` now include
  `supportedTargets`, derived from existing `NeedsWindow`, `NeedsTarget`, and
  `TargetCapabilities`.
- Catalog export also includes `supportedTargets`, and Markdown marks include a
  `Supports:` marker.
- The derivation only publishes user-visible target kinds:
  - `win32-window`
  - `android-adb`
- Internal `browser-cdp` remains unavailable as product metadata after
  `BrowserTarget` deletion.
- Frontend registry adapts `supportedTargets` into `NodeKindSpec`.
- Node library items show compact platform badges:
  - Windows-only nodes: `Win only` / `仅 Win`
  - Android-only nodes: `Android only` / `仅 Android`
  - Shared target nodes: `Win` and `Android`
- Container service can resolve the editor target kind for a node:
  - nearest upstream target selection when `nodeID` is provided;
  - first main-graph target selection when no node context exists;
  - historical Windows default when no explicit target exists.
- Tools service now routes `OpenScreenPicker` and `PixelAt` through
  `TargetToolAdapter`.
- Win32 adapter wraps the existing screen picker and pixel sampling behavior.
- Android adapter exists as a clear boundary, currently returning explicit
  not-implemented errors for picker preview and pixel sampling.

## Tests

- `go test ./internal/node ./internal/catalog ./internal/services/container ./internal/services/tools -count=1`
- `pnpm -C frontend test src/components/containers/nodeRegistry/adapter.test.ts src/components/containers/nodeRegistry/platformTargets.test.ts`
- `pnpm -C frontend i18n:check`
- `pnpm -C frontend typecheck`

## Next

Build the Android target preview implementation behind the new adapter:

1. Resolve full Android target data for editor tooling, not only target kind.
2. Add an Android preview screenshot endpoint using ADB/controller screenshot.
3. Add an app-owned preview picker route for point/rect/template/color.
4. Convert preview-image coordinates to normalized/android-device coordinates.
5. Use the same preview frame for Android pixel sampling.

