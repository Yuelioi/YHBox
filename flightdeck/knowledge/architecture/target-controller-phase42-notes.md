# Target Controller Upgrade — Phase 42 Notes

## Completed

- Added a filesystem/parser guard for `internal/nodes/all`.
- Switched the container package's broad test setup to the centralized registration import.

## Verification

- `go test ./internal/nodes/all ./internal/services/container -count=1`
- `go test ./...`
- `git diff --check`

## Result

The central registration package now fails tests if a new built-in node package is added without updating `internal/nodes/all/doc.go`. This turns node registration drift into an immediate test failure instead of a late catalog/runtime surprise.
