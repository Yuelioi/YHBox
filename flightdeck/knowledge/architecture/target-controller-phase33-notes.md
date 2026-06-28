# Target Controller Upgrade — Phase 33 Notes

## Completed

- Extended `internal/node/spec_consistency_test.go` to import all node packages, matching catalog coverage.
- Added widget kind validation for backend `WidgetSpec.Kind`.
- Added props shape guards:
  - `dropdown` must have non-empty `options`.
  - `slider` must have numeric `min/max/step`, `min < max`, and `step > 0`.
  - `async-dropdown` must have a non-empty `asyncSource`.
  - `async-dropdown.applyMeta` must map non-empty meta keys to existing non-exec input pins on the same node.

## Verification

- `go test ./internal/node -count=1`
- `pnpm vitest run src/components/containers/nodeRegistry/adapter.test.ts`
- `pnpm vue-tsc --noEmit`

## Note

The frontend adapter vitest still prints the existing `/wails/custom.js` 404 noise after the pass summary. It exits 0 and all 8 adapter tests pass.

## Next Risk

The next guard should focus on target/controller compatibility: nodes with `NeedsWindow`/`NeedsForeground` and active target kinds should fail predictably when a backend cannot provide required capabilities.
