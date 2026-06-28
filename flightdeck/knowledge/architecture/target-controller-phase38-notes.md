# Target Controller Upgrade — Phase 38 Notes

## Completed

- Added runtime no-fallback contract tests for stale-window risk:
  - `InputAdapter` with an Android active target and an older Win32 active window must fail through the controller factory path, not click the previous HWND.
  - `CaptureAdapter` must not screenshot the previous HWND once a non-Win32 active target exists.
  - `VisionAdapter.DetectColor` must not use the Win32 capture backend after switching to a non-Win32 target.

## Verification

- `go test ./internal/services/container/runtime -run "Test(InputAdapter_ActiveTargetDoesNotFallbackToPreviousWindow|CaptureAdapter_ActiveTargetDoesNotFallbackToPreviousWindow|VisionAdapter_ActiveTargetDoesNotFallbackToPreviousWindowCapture)" -count=1`
- `go test ./internal/services/container/runtime -count=1`

## Result

The AE/main-window screenshot-picking class of regression is now pinned at the runtime adapter boundary: once `ActiveTarget` is Android/CDP/mock, adapters cannot silently use a stale HWND fallback.

## Next Risk

Continue broad hardening with node catalog/i18n completeness and runtime graph migration checks, then run full Go and frontend gates before a larger review checkpoint.
