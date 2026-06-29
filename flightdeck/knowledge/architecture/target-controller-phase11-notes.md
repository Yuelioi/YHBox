# Target / Controller Phase 11 Notes

SUMMARY: Phase 11 routes relative mouse movement through Win32Controller as a raw delta action
READ WHEN: Debugging MouseMoveRel traces / changing raw-input camera movement / reviewing input controller coverage
RECHECK WHEN: `Win32Controller.MoveRelative`, runtime `inputAdapter.MouseMoveRel`, or backend raw mouse movement changes

---

Phase 11 migrates relative mouse movement:

- `RelativeMoveRequest` carries `Dx`, `Dy`, and `DurationMs`.
- `Win32Controller.MoveRelative` records a `move-relative` action and delegates to backend `MouseMoveRel`.
- `inputAdapter.MouseMoveRel` now routes through the controller.
- `runtimeWin32Input` delegates relative movement to the existing `pkg/input.Backend`.

Design constraint:

- `move-relative` intentionally has no coordinate steps. It is raw delta movement, not a normalized point transform.

Operational effect:

- `MouseMoveRel` nodes produce controller action traces when dispatched through the framework.
- Those traces include Phase 8 source metadata.
- Current runtime `InputService` methods that map to Win32 input are now routed through `Win32Controller`.

Still not covered:

- Screenshot/capture routing through controller.
- UI trace viewer and persistence.
- Browser/Android controller implementations.
