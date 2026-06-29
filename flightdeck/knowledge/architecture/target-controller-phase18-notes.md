# Target / Controller Phase 18 Notes

SUMMARY: Phase 18 adds code-level controller backend capability profiles
READ WHEN: Choosing or validating a controller backend for Win32, Browser CDP, Android ADB, mock, or replay targets
RECHECK WHEN: Adding a controller backend, target kind, coordinate space, or capability

---

Phase 18 creates a reusable backend matrix in `internal/automation/controller`:

- `BackendKind` constants: `win32`, `android-adb`, `browser-cdp`, `debug-replay`, `mock`.
- `BackendProfile` describes supported target kinds, coordinate spaces, and default capabilities.
- Lookup helpers:
  - `Profile(kind)`
  - `Profiles()`
  - `ProfilesForTargetKind(kind)`
  - `DefaultProfileForTargetKind(kind)`
- `Profiles()` returns stable order for future UI/docs generation.

Capability boundary:

- Win32 keeps screenshot/click/move/scroll/key chord/key state/text.
- Android ADB advertises screenshot/click/move/scroll/text/app lifecycle, but not key-state.
- Browser CDP advertises screenshot/click/move/scroll/key chord/key state/text.
- Debug replay is screenshot-only.
- Mock advertises full input/screenshot capabilities for tests.

Verification:

- `go test ./internal/automation/controller -count=1`

Still not covered:

- Concrete Android ADB controller transport.
- Concrete Browser CDP controller transport.
- Runtime/UI selection based on profiles.
