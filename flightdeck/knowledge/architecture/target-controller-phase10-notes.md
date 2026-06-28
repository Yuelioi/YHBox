# Target / Controller Phase 10 Notes

SUMMARY: Phase 10 routes mouse hold and drag input through Win32Controller with trace/source metadata
READ WHEN: Debugging MouseHold/Swipe traces / changing drag semantics / planning pointer backend abstractions
RECHECK WHEN: `Win32Controller.MouseDown`, `MouseUp`, `Drag`, runtime `inputAdapter`, or backend drag APIs change

---

Phase 10 migrates hold and drag input:

- `Win32Controller.MouseDown` records a `mouse-down` action with one normalized coordinate step.
- `Win32Controller.MouseUp` records a `mouse-up` action without coordinate steps because the backend API releases by button.
- `Win32Controller.Drag` records a `drag` action with begin and end normalized coordinate steps.
- `inputAdapter.MouseDown`, `MouseUp`, and `Drag` now delegate through the controller.
- `runtimeWin32Input` delegates the expanded controller input interface to existing `pkg/input.Backend` methods.

Operational effect:

- `MouseHoldStart`, `MouseHoldStop`, and `Swipe` produce controller action traces when dispatched through the framework.
- Those traces include Phase 8 source metadata.
- Backend behavior remains unchanged; drag is still delegated to the selected backend primitive.

Still not covered:

- `MouseMoveRel` controller routing.
- Rich drag trajectory trace beyond begin/end steps.
- UI trace viewer and persistence.
