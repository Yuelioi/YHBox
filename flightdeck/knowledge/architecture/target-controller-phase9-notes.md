# Target / Controller Phase 9 Notes

SUMMARY: Phase 9 routes runtime TypeText through Win32Controller.Text and records text input traces
READ WHEN: Debugging InputText traces / changing text input behavior / planning browser or Android text policy
RECHECK WHEN: `inputAdapter.TypeText`, `Win32Controller.Text`, or `pkg/input` text injection changes

---

Phase 9 migrates text input routing:

- `inputAdapter.TypeText` now delegates through `Win32Controller.Text`.
- `Win32Controller.Text` records a `text` action and then calls the selected backend `TypeText`.
- Framework-dispatched `InputText` nodes inherit Phase 8 source metadata on the `text` trace.

Operational effect:

- `InputText` still uses the configured runtime backend (`sendinput`, `postmessage`, etc.).
- A successful `InputText` node emits one `text` trace record with container/node/kind/in-pin source.
- Text injection policy is still owned by `pkg/input`; this phase only moves the trace/control boundary.

Still not covered:

- IME strategy, clipboard paste fallback, browser DOM text, Android text.
- Drag, MouseDown, MouseUp controller routing.
- UI trace viewer and persistence.
