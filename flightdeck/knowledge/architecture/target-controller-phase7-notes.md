# Target / Controller Phase 7 Notes

SUMMARY: Phase 7 routes runtime Scroll through Win32Controller and records a minimal coordinate step
READ WHEN: Continuing pointer migration / debugging scroll traces / planning drag or trace metadata support
RECHECK WHEN: `Win32Controller.Scroll`, `inputAdapter.Scroll`, or scroll axis semantics change

---

Phase 7 migrates scroll input:

- `Win32Controller.Scroll` records a `scroll` action with one `CoordinateStep`.
- The coordinate step records normalized input mapped to `window-client` target space.
- `inputAdapter.Scroll` now delegates through `Win32Controller.Scroll`.
- Existing `pkg/input.Backend` still receives the same hwnd, normalized ratios, notches, and horizontal flag.

Operational effect:

- `Scroll` node behavior is unchanged externally.
- Trace now covers KeyDown, KeyUp, Click, MoveTo, and Scroll runtime input paths.

Still not covered:

- Drag, MouseDown, MouseUp, TypeText, Screenshot.
- Node id / pin id trace metadata.
- Pixel-to-normalized conversion as a recorded coordinate step.
- UI trace viewer and persistence.
