# Target Controller Upgrade — Phase 30 Notes

## Completed

- Wrapped `browsercdp.ClientProvider` cached websocket clients in a managed `controller.CDPClient`.
- On CDP call error, the wrapper invalidates the cached BrowserID entry if it still points to the failed websocket.
- Stale websocket close is best-effort.
- The next `ClientForTarget` call rediscovers/redials instead of reusing the broken connection.
- No automatic replay of failed CDP commands was added, because input commands may not be idempotent.

## Verification

- `go test ./internal/services/browsercdp ./internal/services/container/runtime ./internal/automation/controller -count=1`

## Boundary

The command that observes the websocket failure still fails. This is intentional: retrying a click/key/text command could double-fire if Chrome processed it before the response path broke.

## Next Risk

Browser automation still depends on the user launching a browser with remote debugging enabled. The next release-facing improvement is either a browser launcher/attach helper or a general hardening pass across node specs and translations.
