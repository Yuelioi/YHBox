# Target Controller Upgrade — Phase 58 Notes

## Completed

- Added `TestInputAdapter_PropagatesControllerFactoryError`.
- Added `TestVisionAdapter_PropagatesControllerFactoryError`.
- Extended the runtime fake controller factory with an error branch.

## Verification

- `go test ./internal/services/container/runtime -count=1`

## Result

Runtime input and vision paths now explicitly preserve controller factory errors for active non-Win32 targets without falling back to stale HWND backends.
