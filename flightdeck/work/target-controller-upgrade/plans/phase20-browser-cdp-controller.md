# Phase 20 — Browser CDP Controller Skeleton

## Goal

Add a testable Browser CDP controller implementation behind the existing controller interfaces, without opening a real WebSocket connection yet.

## Scope

- Add `BrowserCDPController` for `target.KindBrowserCDP`.
- Inject a `CDPClient` interface so tests do not require Chrome.
- Implement:
  - screenshot via `Page.captureScreenshot`;
  - click/move/drag/scroll via `Input.dispatchMouseEvent`;
  - key chord/key down/key up via `Input.dispatchKeyEvent`;
  - text via `Input.insertText`.
- Convert normalized coordinates to browser viewport pixels using target resolution.
- Trace each action through the existing recorder.

## Non-goals

- CDP WebSocket discovery / connection lifecycle.
- Runtime routing to Browser CDP.
- DOM selector actions.

## Verification

- `go test ./internal/automation/controller -count=1`
- `go test ./internal/automation/... -count=1`
