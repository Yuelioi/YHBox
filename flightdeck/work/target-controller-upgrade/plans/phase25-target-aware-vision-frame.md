# Phase 25 — Target-Aware Vision Frame Source

## Goal

Make vision nodes acquire frames from the active automation target instead of assuming a Win32 HWND.

## Scope

- Keep Win32 vision frame cache behavior for existing workflows.
- Add a target-aware live screenshot path for non-Win32 targets through `controllerForActiveTarget`.
- Route template/color/blob/QR/grid/signature vision helpers through a shared frame helper.
- Add tests proving Android active targets can feed vision frame acquisition through an injected controller factory.

## Non-goals

- No matcher algorithm changes.
- No target-aware frame cache for Android/CDP yet.
- No Android device discovery or CDP client lifecycle yet.

## Verification

- `go test ./internal/services/container/runtime -run "TestVisionAdapter|TestDetectColor|TestDecodeQR|TestInputAdapter_ClickRoutesThroughInjectedControllerFactory" -count=1`
- `go test ./internal/services/container/runtime -count=1`
- `go test ./internal/nodes/detect ./internal/nodes/image -count=1`

