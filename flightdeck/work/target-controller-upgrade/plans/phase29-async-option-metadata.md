# Phase 29 — Async Option Metadata Apply

## Goal

Let async dropdown selections carry structured metadata and apply it to sibling node inputs without bespoke inspector sections.

## Scope

- Extend `node.EnumOption` with optional `Meta map[string]any`.
- Extend `node.AsyncDropdownProps` with optional `ApplyMeta map[string]string` where key = option meta key, value = target input pin.
- Preserve backward compatibility for static dropdowns and async sources that only return value/label.
- Populate metadata for:
  - Android ADB devices: display name, width, height
  - Browser CDP targets: title/name, URL, websocket debugger URL
- Update frontend registry adapter to pass `applyMeta`.
- Update `PinInput` to emit selected async option metadata.
- Update `NodeInspector` to apply metadata to sibling literal fields in one config update.

## Non-goals

- No disabled option rows or status badges.
- No browser viewport discovery through CDP.
- No manual refresh button; existing async reload-on-input-change remains.

## Verification

- Go tests for Android/Browser async source metadata and node widget props.
- Frontend adapter test for `applyMeta`.
- `pnpm vue-tsc --noEmit`
- focused Go tests:
  - `go test ./internal/node ./internal/services/androidadb ./internal/services/browsercdp ./internal/nodes/system -count=1`
