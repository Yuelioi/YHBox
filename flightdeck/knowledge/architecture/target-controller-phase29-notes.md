# Target Controller Upgrade — Phase 29 Notes

## Completed

- Extended `node.EnumOption` with optional `Meta`.
- Extended `node.AsyncDropdownProps` with optional `ApplyMeta`.
- Android ADB async options now carry:
  - `name`
  - `width`
  - `height`
- Browser CDP async options now carry:
  - `name`
  - `url`
  - `webSocketDebuggerUrl`
- `AndroidTarget.Serial` maps metadata into `Name`, `Width`, and `Height`.
- `BrowserTarget.BrowserID` maps metadata into `Name` and `WebSocketURL`.
- Frontend node registry preserves `applyMeta`.
- `PinInput` emits selected async option metadata.
- `NodeInspector` applies metadata to sibling `config.literal` fields in one config update.

## Boundary

This is deliberately generic: no Android- or Browser-specific inspector logic was added. Any future async dropdown can opt into metadata application through `ApplyMeta`.

Manual entry remains supported. Metadata is applied only when the selected value matches a loaded async option.

## Verification

- `go test ./internal/node ./internal/services/androidadb ./internal/services/browsercdp ./internal/nodes/system -count=1`
- `go test ./internal/services/browsercdp ./internal/services/androidadb ./internal/nodes/system ./internal/services/container/runtime ./internal/automation/controller ./internal/catalog . -count=1`
- `pnpm vitest run src/components/containers/nodeRegistry/adapter.test.ts`
- `pnpm vue-tsc --noEmit`

The focused vitest run still prints the existing `127.0.0.1:3000` connection-refused aggregate after a passing summary and exits 0.

## Next Risk

CDP stale-client handling is still basic. If Chrome restarts or a tab closes, the cached websocket client may fail on the next call; the next layer should invalidate and rediscover on websocket errors.
