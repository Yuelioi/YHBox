# Target Controller Upgrade — Phase 25 Notes

## Completed

- Added a shared `visionAdapter.captureFrame` helper.
- Win32 vision paths keep the existing HWND frame cache.
- Non-Win32 active targets capture through `controllerForActiveTarget(... Capture: true)` and `controller.Screenshotter`.
- Migrated these vision paths to the helper:
  - `Match` / `WaitMatch`
  - `MatchAll`
  - `DetectColor`
  - `DetectColorHSV`
  - `DetectColorBlobs`
  - `ROIColorScan`
  - `DualBarTrack`
  - `GridSignature`
  - `FindColorSignature`
  - `DecodeQR`
- Added a runtime test proving Android active target screenshot can feed `DetectColor` without a Win32 capture backend.

## Boundary

This phase makes frame acquisition target-aware. It does not add a target-aware frame cache for Android/CDP, and it does not change matcher algorithms.

## Verification

- `go test ./internal/services/container/runtime -run "TestVisionAdapter|TestDetectColor|TestDecodeQR|TestInputAdapter_ClickRoutesThroughInjectedControllerFactory" -count=1`
- `go test ./internal/services/container/runtime -count=1`
- `go test ./internal/nodes/detect ./internal/nodes/image -count=1`

## Next Risk

Android target selection still requires manual serial/resolution entry. The next slice should add ADB discovery so UI/node config can list connected devices and their current resolution.

