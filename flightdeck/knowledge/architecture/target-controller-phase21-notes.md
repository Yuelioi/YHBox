# Target / Controller Phase 21 Notes

SUMMARY: Phase 21 adds active automation target state to runtime
READ WHEN: Routing runtime services to non-Win32 controllers or changing WindowTarget behavior
RECHECK WHEN: `RuntimeContext.SetActiveWindow`, `SetActiveTarget`, input/capture adapters, or target selection nodes change

---

Phase 21 introduces active target state in `RuntimeContext`:

- `SetActiveWindow(wh)` still updates sticky `WindowHandle`, and now also sets active `target.Target` via `windowHandleToTarget(wh)`.
- `SetActiveTarget(tg)` / `ActiveTarget()` allow future Android/Browser target-selection nodes to set non-Win32 targets without pretending they are HWNDs.
- Input and capture adapters now construct their Win32 controller from active target, with fallback to `WindowHandle` for old tests/paths.
- Input/capture adapters explicitly reject non-Win32 active targets until runtime routing to Android/Browser controllers is implemented.

Verification:

- `go test ./internal/services/container/runtime -run "TestRuntimeContext|TestInputAdapter|TestCaptureAdapter|TestWindowHandleToTarget" -count=1`
- `go test ./internal/services/container/runtime -count=1`

Still not covered:

- Android/Browser target-selection nodes.
- Runtime controller factory that chooses Win32 vs Android ADB vs Browser CDP.
- UI for selecting active non-Win32 targets.
