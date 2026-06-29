# Target Controller Upgrade — Phase 31 Notes

## Completed

- Added a catalog drift guard for declared input/output pin labels.
- Added a shared frontend fallback label for the standard exec input pin `In`.
- Added shared i18n keys:
  - `common.exec_in_pin` = `执行`
  - `common.exec_in_pin` = `Run`
- `catalog.BuildWithI18n()` now synthesizes the shared exec input label instead of requiring repeated `In` entries on every node.

## Verification

- `go test ./internal/catalog -count=1`
- `pnpm i18n:check`
- `pnpm vue-tsc --noEmit`

## Boundary

Hints remain optional. Forcing every pin to have a hint would produce filler text and reduce signal. The hard contract is node label/description plus every declared pin label.

## Next Risk

The i18n check is now clean and pin label drift is guarded. The next hardening target is node spec/runtime consistency beyond labels: widget props, async source registration, and target/controller capability compatibility.
