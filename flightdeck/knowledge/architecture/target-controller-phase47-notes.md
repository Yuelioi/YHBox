# Target Controller Upgrade — Phase 47 Notes

## Completed

- Added `TestSpecConsistency_ExecOutPinNamingConvention`.
- Updated `node-spec-style.md` to state exec output naming is now test-guarded.

## Verification

- `go test ./internal/node -run TestSpecConsistency -count=1`
- `go test ./internal/node ./internal/catalog -count=1`
- `go test ./...`

## Result

Lowercase exec output names now fail Go tests immediately, except the reserved Switch `default` exit. This closes a long-standing manual-review gap in node spec naming.
