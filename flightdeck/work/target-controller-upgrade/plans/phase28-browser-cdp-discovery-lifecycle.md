# Phase 28 — Browser CDP Discovery and Client Lifecycle

## Goal

Make Browser CDP targets usable end to end for browsers already launched with a remote debugging port.

## Scope

- Add `internal/services/browsercdp`:
  - discover targets from Chrome-compatible `/json` endpoint
  - parse page targets with `id`, `title`, `url`, `webSocketDebuggerUrl`
  - expose async dropdown source `browserCDPTargets`
  - provide a cached CDP client provider keyed by BrowserID
- Add a `BrowserTarget` node:
  - selects a discovered browser page target
  - sets active `target.KindBrowserCDP`
  - carries browser ID, display name, debugger URL, endpoint, and viewport size
- Wire `DefaultControllerFactory` to build `BrowserCDPController` with a live client provider.
- Register the async source in `main.go` and use the same provider for GUI runtime.

## Non-goals

- Do not launch Chrome/Edge automatically.
- Do not add a dedicated frontend browser target panel.
- Do not add rich option metadata or auto-fill sibling fields from dropdown selection.
- Do not change the Browser CDP controller action protocol beyond what is needed for client lifecycle.

## Verification

- Unit-test discovery parsing and async source behavior.
- Unit-test websocket client/provider behavior with an in-process test server if practical.
- Unit-test `BrowserTarget` active target output.
- Unit-test `DefaultControllerFactory` browser wiring.
- Run focused Go tests:
  - `go test ./internal/services/browsercdp ./internal/nodes/system ./internal/services/container/runtime ./internal/automation/controller -count=1`

## Risk

Without launching the browser automatically, users must start Chrome/Edge with a remote debugging port, usually `--remote-debugging-port=9222`. That is acceptable for this phase because it keeps lifecycle small and testable.
