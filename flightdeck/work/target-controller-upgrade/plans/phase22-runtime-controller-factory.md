# Phase 22 — Runtime Controller Factory

## Goal

Let runtime services resolve controllers from the active target instead of hard-coding Win32 in each adapter, while preserving current Win32 behavior.

## Scope

- Add a `RuntimeControllerFactory` injection point to `RuntimeContext`.
- Add runtime helper that:
  - builds a Win32 controller for Win32 active targets using existing input/capture deps;
  - delegates non-Win32 active targets to the injected factory;
  - records traces with the current node source.
- Update input/capture adapters to request `PointerInput`, `KeyboardInput`, or `Screenshotter` capabilities from the resolved controller.
- Add tests proving an injected non-Win32 controller can receive input through `NewInputAdapter`.

## Non-goals

- Build real ADB/CDP connection discovery in runtime.
- Add target-selection nodes.
- Change existing Win32 behavior.

## Verification

- Focused runtime adapter tests.
- Full `go test ./internal/services/container/runtime -count=1`.
