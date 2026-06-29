# Target Controller Upgrade — Phase 34 Notes

## Completed

- Extended `controller.CapabilitySet` with granular pointer capabilities:
  - `mouse-button`
  - `drag`
  - `move-relative`
- Updated backend profiles:
  - Win32: supports mouse button, drag, and relative move.
  - Android ADB: supports drag, but not mouse button hold/release or relative move.
  - Browser CDP: supports mouse button and drag, but not relative move.
- Updated runtime input/capture adapters to check controller capabilities before invoking the action.
- Added regression coverage for Android rejecting unsupported mouse hold and relative movement before any backend call.

## Verification

- `go test ./internal/automation/controller ./internal/services/container/runtime -count=1`
- `go test ./internal/automation/controller -count=1`
- `go test ./internal/services/container/runtime -run TestInputAdapter_RejectsUnsupportedAndroidCapabilitiesBeforeCallingController -count=1`

## Note

Superseded by Phase 35: the full `internal/services/container/runtime` package now passes in about 4.4 seconds in this workspace.

## Next Risk

Target capability compatibility is now explicit for current input actions. The next useful pass is broader runtime and frontend contract hardening.
