# Target / Controller Phase 22 Notes

SUMMARY: Phase 22 adds runtime controller factory resolution from active targets
READ WHEN: Wiring Android/Browser controllers into runtime or changing input/capture adapters
RECHECK WHEN: `RuntimeControllerFactory`, `RuntimeContext.ControllerFactory`, or active target routing changes

---

Phase 22 changes runtime services from direct Win32 construction to active-target controller resolution:

- `RuntimeContext.ControllerFactory` can be injected for non-Win32 targets.
- `controllerForActiveTarget(source, need)`:
  - builds `Win32Controller` for Win32 active targets using existing runtime input/capture deps;
  - delegates non-Win32 active targets to the injected factory;
  - wraps trace recorder with current node source.
- `inputAdapter` now requests `controller.PointerInput` / `controller.KeyboardInput`.
- `captureAdapter` now requests `controller.Screenshotter`.
- Current Win32 behavior is preserved.
- Tests prove an Android active target can route `NewInputAdapter(...).Click(...)` through an injected controller factory.

Verification:

- `go test ./internal/services/container/runtime -run "TestRuntimeContext|TestInputAdapter|TestCaptureAdapter|TestWindowHandleToTarget" -count=1`
- `go test ./internal/services/container/runtime -count=1`

Still not covered:

- Real ADB/CDP controller factories in app wiring.
- Android/Browser target-selection nodes.
- UI target discovery/selection.
