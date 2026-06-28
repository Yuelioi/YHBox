# Target / Controller Phase 4 Notes

SUMMARY: Phase 4 routes runtime KeyDown/KeyUp through Win32Controller with runtime trace
READ WHEN: Continuing input-node migration / debugging keyboard trace / planning click or text controller routing
RECHECK WHEN: `inputAdapter`, `Win32Controller`, or `pkg/input.Backend` changes

---

Phase 4 migrates the first runtime action path:

- `inputAdapter.KeyDown` and `inputAdapter.KeyUp` now construct a `Win32Controller`.
- The controller target comes from the current active `WindowHandle`.
- The controller input dependency wraps the existing `pkg/input.Backend`.
- Trace records are written to the current `RuntimeContext`.

Still not migrated:

- Click, move, scroll, drag, text, screenshot.
- Node id / pin id trace metadata.
- Android/browser/CDP controllers.
- UI trace viewer and persistence.

Operational note:

- Existing nodes keep their public behavior and still call `node.InputService`.
- `KeyPress` emits two trace records, `key-down` then `key-up`.
- Backend labels come from `pkg/input.Backend.Name()`.
