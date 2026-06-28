# Phase 19 — Android ADB Controller Skeleton

## Goal

Add a concrete, testable Android ADB controller implementation behind the existing controller interfaces without wiring it into runtime execution yet.

## Scope

- Add `AndroidADBController` for `target.KindAndroidADB`.
- Inject an `ADBRunner` interface so tests do not require a real device.
- Implement:
  - screenshot via `exec-out screencap -p`;
  - click/tap via `shell input tap`;
  - drag via `shell input swipe`;
  - scroll via `shell input swipe`;
  - text via `shell input text`;
  - app start/stop via `monkey` / `am force-stop`.
- Trace each action through the existing recorder.
- Add focused controller tests.

## Non-goals

- Runtime routing to Android.
- Device discovery.
- MaaFramework integration.
- Browser controller.

## Verification

- `go test ./internal/automation/controller -count=1`
