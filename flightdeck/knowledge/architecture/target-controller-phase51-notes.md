# Target Controller Upgrade — Phase 51 Notes

## Completed

- Added `TestSpecConsistency_GeometryDefaultsUseNodeTypesWhenSet`.
- Corrected stale node spec knowledge: Point/Rect spec defaults should be `node.Point` / `node.Rect`, not JSON-like maps.

## Verification

- `go test ./internal/node -run TestSpecConsistency -count=1`

## Result

Point/Rect defaults are now documented and guarded according to the actual runtime default path.
