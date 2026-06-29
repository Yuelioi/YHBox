# Phase 73 — Android Editor Picker Capture

Date: 2026-06-29

## Goal

Let editor tooling reuse the existing screen picker for Android ADB targets by routing the picker screenshot source through the active editor target instead of hard-binding it to Win32 HWND capture.

## Research Input

Before implementation, `blue_archive_auto_script` was cloned under `flightdeck/references/` and summarized in `flightdeck/references/blue_archive_auto_script-findings.md`.

Useful design takeaways:
- Keep screenshot/control backends behind runtime adapters.
- Start with generic ADB screenshot/input before emulator-specific fast paths.
- Treat emulator discovery as a separate helper, not as a picker responsibility.
- Avoid copying GPL-3.0 code.

## Implementation

- Added full editor target resolution in `container.Service`:
  - `ResolveEditorTargetForNode(containerID,nodeID)` returns `target.Target`.
  - `ResolveEditorTargetKindForNode` now delegates to the full resolver.
- Added Android target config parsing for editor tooling:
  - serial
  - display name
  - width/height
- Updated `templateCaptureAdapter`:
  - Win32 keeps the existing HWND capture path.
  - Android uses `AndroidADBController.Screenshot` and returns PNG bytes to the existing asset capture RPC.
- Updated tools picker routing:
  - Android picker now opens the same `ScreenPickerView`.
  - The view already calls `assets.capture(containerID,nodeID)`, so point/rect/template/color modes receive Android screenshots through the new capture route.
  - `PixelAt` remains explicitly unsupported for Android because the current MouseHUD API has no Android coordinate input; it is cursor-under-window semantics.
- Improved Android ADB discovery:
  - If `adb devices -l` has no online `device`, `androidadb.Service` tries a small set of common emulator ADB addresses before listing again.
  - Initial common addresses cover MuMu 12, MuMu/common emulator, LDPlayer/common default, Nox, and MEmu.
- Added bundled Windows ADB runtime support:
  - `internal/adbexec` resolves `YOTTA_ADB_PATH` / `YHFISH_ADB_PATH`, then bundled `platform-tools`, then PATH `adb`.
  - Android controller and Android discovery share the same executable resolver.
  - Windows build copies `build/windows/platform-tools` into `bin/platform-tools`; NSIS installs it beside the app.

## Verification

Targeted tests added/updated:
- `internal/services/container/window_resolve_for_node_test.go`
- `internal/services/tools/target_tool_test.go`
- `wire_templates_test.go`
- `internal/services/androidadb/discovery_test.go`
- `internal/adbexec/executable_test.go`

Targeted commands passed:
- `go test ./internal/services/container -run 'TestEditorTarget(Kind|ForNode)'`
- `go test ./internal/services/tools -run 'Test(AndroidTargetToolAdapter|TargetToolRouter|ServiceOpenScreenPicker|ServicePixelAt)'`
- `go test . -run TestTemplateCaptureAdapter_CaptureAndroidTargetUsesADBScreenshot`
- `go test ./internal/services/androidadb`
- `go test ./internal/adbexec`
- `task windows:build:adb`
- `go test ./...`
- `pnpm -C frontend typecheck`
- `pnpm -C frontend test`

## Remaining

- Add a user-facing Android device test flow with MuMu:
  1. choose `AndroidTarget.Serial`
  2. use screenshot point picker on an Android-backed node
  3. use screenshot rectangle picker
  4. save or recapture a template from the Android screenshot
- Add Android input tool preview only after the generic ADB path is proven.
- Do not add MuMu IPC / Nemu fast path until generic ADB support is stable and there is a real performance need.
