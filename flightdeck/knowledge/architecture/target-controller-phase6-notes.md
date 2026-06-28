# Target / Controller Phase 6 Notes

SUMMARY: Phase 6 routes runtime MoveTo through Win32Controller and adds a minimal coordinate-step trace for move
READ WHEN: Continuing pointer migration / debugging click movement traces / planning coordinate conversion or drag support
RECHECK WHEN: `Win32Controller.Move`, `inputAdapter.MoveTo`, or coordinate trace semantics change

---

Phase 6 migrates pointer movement:

- `Win32Controller.Move` records a `move` action with one `CoordinateStep`.
- The coordinate step records normalized input mapped to `window-client` target space.
- `inputAdapter.MoveTo` now delegates through `Win32Controller.Move`.
- Existing `pkg/input.Backend` still receives the same hwnd and normalized ratios.

Operational effect:

- A normal `ClickAt` path can now emit `move` then `click` trace records.
- `ClickAt` still resolves point units before calling `InputService`.
- Backend labels still come from `pkg/input.Backend.Name()`.

Still not covered:

- Pixel-to-normalized conversion is not recorded as a controller coordinate step yet.
- Drag, mouse down/up, scroll, text, screenshot remain outside controller routing.
- Trace records still lack node id / pin id metadata.
