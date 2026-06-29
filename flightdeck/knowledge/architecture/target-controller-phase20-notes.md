# Target / Controller Phase 20 Notes

SUMMARY: Phase 20 adds a testable Browser CDP controller skeleton
READ WHEN: Extending browser automation, CDP transport, or browser target routing
RECHECK WHEN: CDP method mapping, browser viewport coordinates, or controller runtime routing changes

---

Phase 20 adds `BrowserCDPController` under `internal/automation/controller`:

- Accepts only `target.KindBrowserCDP`.
- Uses injected `CDPClient` so tests do not require Chrome or a WebSocket.
- Implements:
  - screenshot: `Page.captureScreenshot` with base64 PNG decode;
  - click/move/drag/scroll: `Input.dispatchMouseEvent`;
  - key chord/down/up: `Input.dispatchKeyEvent`;
  - text: `Input.insertText`.
- Normalized points require target resolution and clamp to browser viewport bounds.
- Actions emit existing `automation/trace.ActionRecord` records.
- Nil CDP clients return an explicit error instead of panicking.

Verification:

- `go test ./internal/automation/controller -count=1`
- `go test ./internal/automation/... -count=1`

Still not covered:

- CDP WebSocket discovery and connection lifecycle.
- Runtime routing to Browser CDP targets.
- DOM selector / accessibility tree actions.
