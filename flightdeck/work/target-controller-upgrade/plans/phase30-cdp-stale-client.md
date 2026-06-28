# Phase 30 — CDP Stale Client Invalidation

## Goal

Avoid permanently reusing a dead Browser CDP websocket client after Chrome restarts, a tab closes, or the websocket is otherwise broken.

## Scope

- Wrap cached CDP clients returned by `browsercdp.ClientProvider`.
- On CDP call error, invalidate that cached client if it is still the active entry for the BrowserID.
- Close the stale websocket best-effort.
- Make the next `ClientForTarget` call rediscover/redial.
- Do not automatically retry the failed CDP command, because input commands may not be idempotent.

## Non-goals

- No automatic browser launch.
- No frontend stale-target warning UI.
- No command replay.

## Verification

- Unit test that a failed call invalidates the cached client and subsequent `ClientForTarget` dials a new websocket.
- `go test ./internal/services/browsercdp ./internal/services/container/runtime ./internal/automation/controller -count=1`
