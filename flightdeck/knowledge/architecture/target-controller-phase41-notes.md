# Target Controller Upgrade — Phase 41 Notes

## Completed

- Switched app startup to the centralized built-in node registration package.
- Switched the full runtime dispatch test to the same centralized registration package.
- Updated catalog documentation to point callers at `internal/nodes/all`.

## Verification

- `go test ./...`
- `git diff --check`

## Result

The shipped app, catalog tooling, MCP tests, CLI helpers, and full runtime dispatch tests now share one built-in node registration surface. Adding a new built-in node package should require updating `internal/nodes/all` once, then existing catalog/i18n/runtime guards can cover it consistently.
