# Target Controller Upgrade — Phase 28 Notes

## Completed

- Added `internal/services/browsercdp`.
- Discovery:
  - fetches Chrome-compatible `/json`
  - parses page targets
  - filters non-page and targets without `webSocketDebuggerUrl`
- Added `browserCDPTargets` NodeService async source.
- Added a serialized websocket JSON-RPC CDP client.
- Added a cached `ClientProvider` keyed by BrowserID.
- Added `BrowserTarget` node:
  - async dropdown for BrowserID
  - Endpoint input with default `http://127.0.0.1:9222`
  - optional advanced WebSocketURL override
  - Width/Height viewport metadata
  - sets active `target.KindBrowserCDP`
- Wired GUI runtime `DefaultControllerFactory` with the Browser CDP provider.
- Preserved the explicit "client is not wired" error when a runtime factory has no provider.

## Boundary

This phase does not launch Chrome/Edge. The browser must already be running with a remote debugging port, for example `--remote-debugging-port=9222`.

`EnumOption` still carries only value/label, so selecting a browser target does not auto-fill title, websocket URL, or viewport size. `BrowserTarget` can rediscover the websocket URL from Endpoint + BrowserID at runtime when WebSocketURL is empty.

MCP micro-runs still use a plain `DefaultControllerFactory{}` in `internal/services/mcpserver/tools_exec.go`; GUI container runs are wired. That is acceptable until MCP grows target selection beyond its current active-window path.

## Verification

- `go test ./internal/services/browsercdp ./internal/nodes/system ./internal/services/container/runtime ./internal/automation/controller ./internal/catalog . -count=1`

## Next Risk

The next useful slice is UX and robustness around target metadata:

- richer async option metadata or a BrowserTarget inspector affordance to fill Width/Height/WebSocketURL from discovery
- optional browser launcher / attach guidance
- CDP connection invalidation when a tab closes or Chrome restarts
