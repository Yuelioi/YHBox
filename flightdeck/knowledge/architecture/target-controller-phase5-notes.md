# Target / Controller Phase 5 Notes

SUMMARY: Phase 5 routes runtime Click through Win32Controller with runtime trace; MoveTo and drag remain unchanged
READ WHEN: Continuing pointer input migration / debugging click trace / planning coordinate conversion work
RECHECK WHEN: `inputAdapter.Click`, `ClickAt`, `Win32Controller.Click`, or point coordinate handling changes

---

Phase 5 migrates the second runtime action path:

- `inputAdapter.Click` now delegates through `Win32Controller.Click`.
- The controller target comes from the current active `WindowHandle`.
- The existing `pkg/input.Backend` still receives the same hwnd, normalized ratios, button, and duration.
- Runtime trace records a `click` action with backend label from `pkg/input.Backend.Name()`.

Important boundary:

- `ClickAt` still resolves node points before calling `InputService`.
- `ClickAt` still calls `MoveTo` before `Click` when appropriate.
- This phase does not migrate `MoveTo`, `MouseDown`, `MouseUp`, `Drag`, or `Scroll`.
- Trace does not yet include node id / pin id metadata.

Next phase candidates:

- Migrate `MoveTo` plus coordinate-step tracing.
- Migrate `Scroll`.
- Add explicit shortcut/chord service support instead of modeling shortcuts as separate KeyDown/KeyUp nodes.
